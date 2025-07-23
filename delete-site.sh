#!/bin/bash
set -e

cd "$(dirname "$0")"

if [ -z "$1" ]; then
  echo "Usage: $0 <sitename>"
  exit 1
fi

SITENAME="$1"
TARGET_DIR="wordpress-${SITENAME}"
NGINX_CONF="nginx-config/${SITENAME}.conf"

if [ ! -d "$TARGET_DIR" ]; then
  echo "[✖] Site directory '$TARGET_DIR' does not exist."
  exit 1
fi

read -p "Are you sure you want to delete the site '${SITENAME}'? This will remove all data. (y/N): " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 0
fi

echo "[−] Stopping and removing containers..."
docker compose -f "$TARGET_DIR/docker-compose.yml" down --volumes

echo "[−] Removing Nginx config..."
rm -f "$NGINX_CONF"

echo "[↻] Reloading Nginx..."
docker exec wpp-deployer-nginx nginx -s reload

echo "[−] Deleting site directory..."
sudo rm -rf "$TARGET_DIR"

echo "[✔] Site '$SITENAME' deleted successfully."

