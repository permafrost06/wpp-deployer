#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <sitename> <plugin-zip-path>"
  exit 1
fi

sitename="$1"
plugin_zip_path="$2"

# Create directory
mkdir -p "/var/www/wordpress/$sitename"
cd "/var/www/wordpress/$sitename" || exit

# Download WordPress core
wp core download

# Create wp-config.php
wp config create --dbname="$sitename" --dbuser=wordpress_user --dbpass=password

# Create the database
wp db create

# Install WordPress
wp core install --url="rshlog.com/$sitename" --title="$sitename" --admin_user=admin --admin_password=admin --admin_email=admin@rshlog.com

# Install plugin
wp plugin install "$plugin_zip_path" --activate