#!/bin/bash
set -e

# Ensure script is run from wpp-deployer root
cd "$(dirname "$0")"

if [ -z "$1" ]; then
  echo "Usage: $0 <sitename>"
  exit 1
fi

SITENAME="$1"
DOMAIN="${SITENAME}.nshlog.com"
TARGET_DIR="wordpress-${SITENAME}"
NGINX_SNIPPET="nginx-config/${SITENAME}.conf"

if [ -d "$TARGET_DIR" ]; then
  echo "Site '$SITENAME' already exists."
  exit 1
fi

echo "[+] Creating site directory..."
cp -r site-template "$TARGET_DIR"

echo "[+] Replacing placeholders..."
sed -i "s/{{sitename}}/${SITENAME}/g" "$TARGET_DIR/docker-compose.yml"
sed "s/{{sitename}}/${SITENAME}/g" site-template/nginx-snippet.conf > "$NGINX_SNIPPET"

echo "[+] Starting containers for '$SITENAME'..."
docker compose -f "$TARGET_DIR/docker-compose.yml" up -d

echo "[+] Reloading nginx..."
docker exec wpp-deployer-nginx nginx -s reload

# echo "[+] Installing WordPress..."
# docker compose -f "$TARGET_DIR/docker-compose.yml" run --rm wpcli core install \
#   --url="http://${DOMAIN}" \
#   --title="${SITENAME}" \
#   --admin_user=admin \
#   --admin_password=adminpass \
#   --admin_email=admin@${DOMAIN}

echo "[✔] Site '${DOMAIN}' deployed successfully!"

