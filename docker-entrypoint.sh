#!/bin/sh
set -eu

config_path=${CONFIG_PATH:-/var/lib/opencode2api/config.json}
listen_address=${LISTEN_ADDRESS:-0.0.0.0:8080}

if [ "$#" -gt 0 ]; then
    exec "$@"
fi

# Docker bind-mounts a non-existent host FILE as an empty directory.
# The mount point itself is busy and cannot be removed from inside the
# container, so point the app at a writable sibling copy seeded from the
# example config instead of exiting.
if [ -e "$config_path" ] && [ ! -f "$config_path" ]; then
    config_path="$(dirname "$config_path")/config.local.json"
    printf '%s\n' "Config mount is a directory; using $config_path instead."
fi

if [ ! -f "$config_path" ]; then
    mkdir -p "$(dirname "$config_path")"
    cp /app/config.example.json "$config_path"
    printf '%s\n' "config.json not found; created $config_path from the example. Edit ./config.json and restart."
fi

exec /app/opencode2api -config "$config_path" -listen "$listen_address"
