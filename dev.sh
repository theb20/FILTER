#!/usr/bin/env bash
# Lance TOUT l'environnement local : backend (PostgreSQL, NATS, identity, gateway) + site (Vite) + backoffice (Next.js).
#
#   ./dev.sh up        démarre tout
#   ./dev.sh status    état de chaque brique
#   ./dev.sh logs      dernières lignes des journaux
#   ./dev.sh mail      « e-mails » de développement (liens d'invitation du personnel, vérifications d'adresse)
#   ./dev.sh down      arrête tout (les données sont conservées)
#   ./dev.sh admin EMAIL "NOM"   crée le PREMIER super-admin (mot de passe saisi au terminal)
#
#   Site        http://localhost:5173     Backoffice  http://localhost:3100     API (gateway)  http://127.0.0.1:8080
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
# Secrets et réglages locaux : backend/.env (ignoré par git). Une variable déjà exportée dans le terminal reste prioritaire.
. "$ROOT/backend/scripts/load-env.sh"; ff_load_env "$ROOT/backend"
D="${FF_DEV_DIR:-$HOME/.filterfinder}"
SITE_PORT=5173; BO_PORT=3100
mkdir -p "$D"/{log,run}

running() { [ -f "$D/run/$1.pid" ] && kill -0 "$(cat "$D/run/$1.pid")" 2>/dev/null; }

start_web() { # nom dossier commande... ; port
  local name="$1" dir="$2" port="$3"; shift 3
  if running "$name"; then return; fi
  if nc -z 127.0.0.1 "$port" 2>/dev/null; then echo "⚠ le port $port est déjà utilisé : $name non démarré (libérez-le puis relancez)"; return; fi
  (cd "$ROOT/$dir" && [ -d node_modules ] || npm install --silent)
  (cd "$ROOT/$dir" && nohup "$@" >"$D/log/$name.log" 2>&1 </dev/null & echo $! > "$D/run/$name.pid")
}

case "${1:-}" in
  up)
    "$ROOT/backend/scripts/dev-local.sh" up
    start_web site frontend "$SITE_PORT" npx vite
    start_web backoffice backoffice "$BO_PORT" npm run dev
    sleep 3; "$0" status
    echo; echo "→ Site : http://localhost:$SITE_PORT   → Backoffice : http://localhost:$BO_PORT/login"
    echo "  Premier lancement : ./dev.sh admin votre@email \"Votre Nom\" pour créer le compte administrateur." ;;
  down)
    for n in site backoffice; do
      if running "$n"; then pkill -P "$(cat "$D/run/$n.pid")" 2>/dev/null || true; kill "$(cat "$D/run/$n.pid")" 2>/dev/null || true; fi
      rm -f "$D/run/$n.pid"
    done
    "$ROOT/backend/scripts/dev-local.sh" down ;;
  status)
    "$ROOT/backend/scripts/dev-local.sh" status
    for n in "site:$SITE_PORT" "backoffice:$BO_PORT"; do
      name="${n%%:*}"; port="${n##*:}"
      nc -z 127.0.0.1 "$port" 2>/dev/null && echo "$name : http://localhost:$port (prêt)" || echo "$name : ne répond pas (voir $D/log/$name.log)"
    done ;;
  logs) "$ROOT/backend/scripts/dev-local.sh" logs; tail -n 15 "$D/log/site.log" "$D/log/backoffice.log" 2>/dev/null || true ;;
  mail) shift; exec "$ROOT/backend/scripts/dev-local.sh" mail "$@" ;;
  admin) shift; exec "$ROOT/backend/scripts/dev-local.sh" bootstrap "$@" ;;
  *) sed -n '2,12p' "$0"; exit 1 ;;
esac
