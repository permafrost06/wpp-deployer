#!/bin/bash
set -e

cd "$(dirname "$0")"

# Helper
print_help() {
  cat <<EOF
Usage: $0 [up|down] [options] [sitename]

Commands:
  up                Bring up a specific site or all sites
  down              Bring down a specific site or all sites

Options (for 'down'):
  -v, --volumes     Remove volumes and disable nginx config
  -h, --help        Show this help message
EOF
}

if [ $# -lt 1 ]; then
  print_help
  exit 1
fi

COMMAND="$1"
shift

INCLUDE_VOLUMES=false

# Parse options for down command
while [[ "$1" =~ ^- ]]; do
  case "$1" in
    -v|--volumes)
      INCLUDE_VOLUMES=true
      shift
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      print_help
      exit 1
      ;;
  esac
done

SITENAME="$1"
NGINX_CHANGED=false

# Reload nginx safely
reload_nginx() {
  echo "[↻] Reloading nginx..."
  if docker exec wpp-deployer-nginx nginx -t; then
    docker exec wpp-deployer-nginx nginx -s reload
  else
    echo "[!] Skipped nginx reload: config test failed"
  fi
}

# Handle one site
run_site() {
  local NAME="$1"
  local DIR="wordpress-$NAME"
  local COMPOSE_FILE="$DIR/docker-compose.yml"
  local NGINX_CONF="nginx-config/$NAME.conf"
  local NGINX_DISABLED="nginx-config/$NAME.conf.disabled"

  case "$COMMAND" in
    up)
      if [ -d "$DIR" ]; then
        echo "[↑] Bringing up $DIR..."
        docker compose -f "$COMPOSE_FILE" up -d

        # Enable nginx config if disabled
        if [ -f "$NGINX_DISABLED" ]; then
          mv "$NGINX_DISABLED" "$NGINX_CONF"
          echo "[+] Enabled nginx config for '$NAME'"
          NGINX_CHANGED=true
        fi
      else
        echo "[!] Site directory '$DIR' not found"
      fi
      ;;
    down)
      if [ -d "$DIR" ]; then
        echo "[↓] Bringing down $DIR..."
        if $INCLUDE_VOLUMES; then
          docker compose -f "$COMPOSE_FILE" down -v
        else
          docker compose -f "$COMPOSE_FILE" down
        fi

        # Disable nginx config
        if [ -f "$NGINX_CONF" ]; then
          mv "$NGINX_CONF" "$NGINX_DISABLED"
          echo "[−] Disabled nginx config for '$NAME'"
          NGINX_CHANGED=true
        fi
      else
        echo "[!] Site directory '$DIR' not found"
      fi
      ;;
    *)
      echo "[!] Unknown command: $COMMAND"
      print_help
      exit 1
      ;;
  esac
}

if [ -n "$SITENAME" ]; then
  run_site "$SITENAME"
else
  echo "[*] No sitename specified, scanning all sites..."
  for dir in wordpress-*; do
    if [ -f "$dir/docker-compose.yml" ]; then
      NAME="${dir#wordpress-}"
      run_site "$NAME"
    fi
  done
fi

# Reload nginx if anything changed
if $NGINX_CHANGED; then
  reload_nginx
fi

