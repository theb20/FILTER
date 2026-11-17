#!/usr/bin/env bash
# Environnement de développement LOCAL persistant, sans Docker : PostgreSQL 17 + NATS JetStream + identity + gateway.
# Les données vivent dans ~/.filterfinder (FF_DEV_DIR) : elles survivent aux redémarrages.
#
#   backend/scripts/dev-local.sh up           démarre tout (premier lancement : crée la base et la clé de chiffrement 2FA)
#   backend/scripts/dev-local.sh bootstrap EMAIL "NOM"   crée le PREMIER super-admin (mot de passe saisi au terminal)
#   backend/scripts/dev-local.sh status | logs | mail | down   (mail : « e-mails » de développement : liens d'invitation…)
#
# Prérequis : brew install postgresql@17 nats-server ; Go.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
D="${FF_DEV_DIR:-$HOME/.filterfinder}"
PG_PORT="${FF_PG_PORT:-5433}"; NATS_PORT="${FF_NATS_PORT:-4223}"
GW_ADDR="${FF_GATEWAY_ADDR:-127.0.0.1:8080}"; ID_ADDR="${FF_IDENTITY_ADDR:-127.0.0.1:9001}"; SRC_ADDR="${FF_SOURCING_ADDR:-127.0.0.1:9002}"; CAT_ADDR="${FF_CATALOG_ADDR:-127.0.0.1:9003}"; CON_ADDR="${FF_CONTENT_ADDR:-127.0.0.1:9004}"
ORIGINS="${ALLOWED_ORIGINS:-http://localhost:3000,http://localhost:5173}"
PGBIN="$(brew --prefix postgresql@17 2>/dev/null)/bin"
DB="filterfinder_identity"; DB_SRC="filterfinder_sourcing"; DB_CAT="filterfinder_catalog"; DB_CON="filterfinder_content"
SOCK="/tmp/ff-pg-$(id -u)"   # chemin court : la limite des sockets Unix est de ~100 caractères
mkdir -p "$D"/{bin,log,run,nats} "$D/secrets" "$SOCK"; chmod 700 "$D/secrets"

url() { echo "postgres://filterfinder@127.0.0.1:${PG_PORT}/${DB}?sslmode=disable"; }
url_src() { echo "postgres://filterfinder@127.0.0.1:${PG_PORT}/${DB_SRC}?sslmode=disable"; }
url_cat() { echo "postgres://filterfinder@127.0.0.1:${PG_PORT}/${DB_CAT}?sslmode=disable"; }
url_con() { echo "postgres://filterfinder@127.0.0.1:${PG_PORT}/${DB_CON}?sslmode=disable"; }

common_env() {
  export ENV=dev STORE=postgres DATABASE_URL="$(url)" NATS_URL="nats://127.0.0.1:${NATS_PORT}"
  export STAFF_TOTP_KEY_FILE="$D/secrets/staff_totp.key" BREACH_CHECK="${BREACH_CHECK:-pwned}" DEV_MAIL_FILE="$D/log/mail.log"
}

up() {
  command -v nats-server >/dev/null || { echo "nats-server introuvable : brew install nats-server"; exit 1; }
  [ -x "$PGBIN/initdb" ] || { echo "PostgreSQL 17 introuvable : brew install postgresql@17"; exit 1; }
  [ -f "$D/pg/PG_VERSION" ] || "$PGBIN/initdb" -D "$D/pg" -U filterfinder --auth=trust -E UTF8 >/dev/null
  # authentification « trust » : acceptable UNIQUEMENT parce que le serveur n'écoute que sur 127.0.0.1 (développement)
  "$PGBIN/pg_isready" -q -h 127.0.0.1 -p "$PG_PORT" || "$PGBIN/pg_ctl" -D "$D/pg" -o "-p $PG_PORT -c listen_addresses=127.0.0.1 -k $SOCK" -l "$D/log/postgres.log" -w start >/dev/null
  "$PGBIN/psql" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder -d postgres -Atc "select 1 from pg_database where datname='$DB'" | grep -q 1 \
    || "$PGBIN/createdb" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder "$DB"
  "$PGBIN/psql" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder -d postgres -Atc "select 1 from pg_database where datname='$DB_SRC'" | grep -q 1 \
    || "$PGBIN/createdb" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder "$DB_SRC"
  "$PGBIN/psql" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder -d postgres -Atc "select 1 from pg_database where datname='$DB_CAT'" | grep -q 1 \
    || "$PGBIN/createdb" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder "$DB_CAT"
  "$PGBIN/psql" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder -d postgres -Atc "select 1 from pg_database where datname='$DB_CON'" | grep -q 1 \
    || "$PGBIN/createdb" -h 127.0.0.1 -p "$PG_PORT" -U filterfinder "$DB_CON"
  [ -f "$D/secrets/staff_totp.key" ] || { (umask 077; openssl rand -base64 32 > "$D/secrets/staff_totp.key"); echo "clé de chiffrement 2FA créée : $D/secrets/staff_totp.key (SAUVEGARDEZ-LA : sans elle, les 2FA enregistrés sont perdus)"; }
  if ! nc -z 127.0.0.1 "$NATS_PORT" 2>/dev/null; then
    nohup nats-server -js -a 127.0.0.1 -p "$NATS_PORT" -sd "$D/nats" >"$D/log/nats.log" 2>&1 </dev/null & echo $! > "$D/run/nats.pid"; sleep 1
  fi
  (cd "$ROOT/services/identity" && go build -o "$D/bin/identity" ./cmd/identity)
  (cd "$ROOT/services/gateway" && go build -o "$D/bin/gateway" ./cmd/gateway)
  (cd "$ROOT/services/sourcing" && go build -o "$D/bin/sourcing" ./cmd/sourcing)
  (cd "$ROOT/services/catalog" && go build -o "$D/bin/catalog" ./cmd/catalog)
  (cd "$ROOT/services/content" && go build -o "$D/bin/content" ./cmd/content)
  common_env
  stop_app
  IDENTITY_GRPC_ADDR="$ID_ADDR" nohup "$D/bin/identity" >"$D/log/identity.log" 2>&1 </dev/null & echo $! > "$D/run/identity.pid"
  for _ in $(seq 1 30); do nc -z "${ID_ADDR%:*}" "${ID_ADDR##*:}" 2>/dev/null && break; sleep 0.5; done
  # Service sourcing. La clé Brave est lue dans l'ENVIRONNEMENT du shell qui lance ce script (export BRAVE_SEARCH_API_KEY=…) :
  # elle n'est écrite dans AUCUN fichier ni journal et n'est jamais transmise en argument.
  if [ -z "${BRAVE_SEARCH_API_KEY:-}" ]; then echo "ℹ BRAVE_SEARCH_API_KEY absente : les appels à Brave échoueront (« secret_missing ») jusqu'à son export, puis ./dev.sh up"; fi
  DATABASE_URL="$(url_src)" SOURCING_GRPC_ADDR="$SRC_ADDR" nohup "$D/bin/sourcing" >"$D/log/sourcing.log" 2>&1 </dev/null & echo $! > "$D/run/sourcing.pid"
  for _ in $(seq 1 30); do nc -z "${SRC_ADDR%:*}" "${SRC_ADDR##*:}" 2>/dev/null && break; sleep 0.5; done
  DATABASE_URL="$(url_cat)" CATALOG_GRPC_ADDR="$CAT_ADDR" nohup "$D/bin/catalog" >"$D/log/catalog.log" 2>&1 </dev/null & echo $! > "$D/run/catalog.pid"
  for _ in $(seq 1 30); do nc -z "${CAT_ADDR%:*}" "${CAT_ADDR##*:}" 2>/dev/null && break; sleep 0.5; done
  DATABASE_URL="$(url_con)" CONTENT_GRPC_ADDR="$CON_ADDR" nohup "$D/bin/content" >"$D/log/content.log" 2>&1 </dev/null & echo $! > "$D/run/content.pid"
  for _ in $(seq 1 30); do nc -z "${CON_ADDR%:*}" "${CON_ADDR##*:}" 2>/dev/null && break; sleep 0.5; done
  ENV=dev SOURCING_ADDR="$SRC_ADDR" CATALOG_ADDR="$CAT_ADDR" CONTENT_ADDR="$CON_ADDR" GATEWAY_ADDR="$GW_ADDR" IDENTITY_ADDR="$ID_ADDR" ALLOWED_ORIGINS="$ORIGINS" COOKIE_SECURE=false nohup "$D/bin/gateway" >"$D/log/gateway.log" 2>&1 </dev/null & echo $! > "$D/run/gateway.pid"
  sleep 1; status
}

stop_app() { for p in gateway content catalog sourcing identity; do [ -f "$D/run/$p.pid" ] && kill "$(cat "$D/run/$p.pid")" 2>/dev/null || true; rm -f "$D/run/$p.pid"; done; }

down() {
  stop_app
  [ -f "$D/run/nats.pid" ] && kill "$(cat "$D/run/nats.pid")" 2>/dev/null || true; rm -f "$D/run/nats.pid"
  "$PGBIN/pg_ctl" -D "$D/pg" -m fast stop >/dev/null 2>&1 || true
  echo "arrêté (données conservées dans $D)"
}

status() {
  for p in identity sourcing catalog content gateway; do
    if [ -f "$D/run/$p.pid" ] && kill -0 "$(cat "$D/run/$p.pid")" 2>/dev/null; then echo "$p : démarré (pid $(cat "$D/run/$p.pid"))"; else echo "$p : arrêté"; fi
  done
  "$PGBIN/pg_isready" -q -h 127.0.0.1 -p "$PG_PORT" && echo "postgresql : prêt (port $PG_PORT)" || echo "postgresql : arrêté"
  nc -z 127.0.0.1 "$NATS_PORT" 2>/dev/null && echo "nats : prêt (port $NATS_PORT)" || echo "nats : arrêté"
  curl -fsS "http://$GW_ADDR/readyz" >/dev/null 2>&1 && echo "gateway : http://$GW_ADDR (prêt)" || echo "gateway : ne répond pas (voir $D/log/gateway.log)"
}

case "${1:-}" in
  up) up ;;
  down) down ;;
  status) status ;;
  logs) tail -n 40 "$D"/log/identity.log "$D"/log/sourcing.log "$D"/log/catalog.log "$D"/log/content.log "$D"/log/gateway.log ;;
  mail) touch "$D/log/mail.log"; tail -n "${2:-30}" "$D/log/mail.log" ;;
  bootstrap)
    [ $# -eq 3 ] || { echo 'usage : dev-local.sh bootstrap EMAIL "NOM"'; exit 1; }
    [ -x "$D/bin/identity" ] || { echo "lancez d'abord : dev-local.sh up"; exit 1; }
    common_env; exec "$D/bin/identity" bootstrap-admin --email "$2" --name "$3" ;;
  *) sed -n '2,10p' "$0"; exit 1 ;;
esac
