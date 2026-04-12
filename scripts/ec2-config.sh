#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
NGINX_CONFIG_SOURCE="$SCRIPT_DIR/nginx.conf"
NGINX_CONFIG_TARGET="/etc/nginx/sites-available/public-decision-tree.conf"
NGINX_ENABLED_TARGET="/etc/nginx/sites-enabled/public-decision-tree.conf"

echo "Updating system packages.."
sudo apt update -y
sudo apt upgrade -y

echo "Installing deployment dependencies.."
sudo apt install -y git golang-go sqlite3 nginx

echo "Preparing application data directory.."
mkdir -p "$APP_DIR/data"

if [ ! -f "$NGINX_CONFIG_SOURCE" ]; then
    echo "Nginx config file not found: $NGINX_CONFIG_SOURCE"
    exit 1
fi

echo "Installing nginx site config.."
sudo cp "$NGINX_CONFIG_SOURCE" "$NGINX_CONFIG_TARGET"
sudo ln -sf "$NGINX_CONFIG_TARGET" "$NGINX_ENABLED_TARGET"
sudo rm -f /etc/nginx/sites-enabled/default

echo "Validating nginx configuration.."
sudo nginx -t

echo "Enabling nginx service.."
sudo systemctl enable nginx
sudo systemctl restart nginx

echo "Configuration complete for $APP_DIR"
