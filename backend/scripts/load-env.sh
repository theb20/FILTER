#!/usr/bin/env bash
# Charge backend/.env dans l'environnement des services lancés par ./dev.sh (le dossier est passé en argument).
#
# Règles tenues ici :
#  - le fichier n'est JAMAIS exécuté : chaque ligne est lue comme « CLÉ=valeur » (une valeur contenant $(…) ou `…`
#    reste du texte) ;
#  - une variable déjà présente dans l'environnement du terminal GAGNE sur le fichier (essai ponctuel :
#    « BRAVE_SEARCH_API_KEY=xxx ./dev.sh up ») ;
#  - le fichier contient des secrets : il est ignoré par git (.gitignore) et doit rester lisible par vous seul.

ff_load_env() {
  local root="$1" file line key value perm
  file="$root/.env"
  [ -f "$file" ] || return 0

  perm="$(stat -f '%OLp' "$file" 2>/dev/null || stat -c '%a' "$file" 2>/dev/null || echo '')"
  case "$perm" in
    ''|600|400) ;;
    *) echo "⚠ $file est lisible par d'autres comptes (droits $perm). Corrigez avec : chmod 600 \"$file\"" ;;
  esac

  local loaded=0 kept=0
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"                               # fichiers enregistrés sous Windows
    line="${line#"${line%%[![:space:]]*}"}"            # espaces de début
    [ -z "$line" ] && continue
    case "$line" in \#*) continue ;; esac
    case "$line" in 'export '*) line="${line#export }" ;; esac
    case "$line" in *=*) ;; *) continue ;; esac        # ligne sans « = » : ignorée
    key="${line%%=*}"; value="${line#*=}"
    key="${key%"${key##*[![:space:]]}"}"               # espaces avant le =
    case "$key" in ''|[0-9]*|*[!A-Za-z0-9_]*) continue ;; esac
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    case "$value" in
      \"*\") value="${value#\"}"; value="${value%\"}" ;;
      \'*\') value="${value#\'}"; value="${value%\'}" ;;
    esac
    [ -z "$value" ] && continue                        # « CLÉ= » : réglage laissé par défaut
    # `printenv` plutôt que l'expansion indirecte de bash : le script reste lisible par sh et zsh.
    if [ -n "$(printenv "$key" 2>/dev/null)" ]; then kept=$((kept + 1)); continue; fi
    export "$key=$value"
    loaded=$((loaded + 1))
  done < "$file"

  [ "$loaded" -gt 0 ] && echo "• .env : $loaded variable(s) chargée(s)${kept:+, $kept déjà définie(s) dans le terminal}"
  return 0
}
