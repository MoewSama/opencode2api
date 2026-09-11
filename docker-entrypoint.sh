#!/bin/sh
set -eu

config_path=${CONFIG_PATH:-/var/lib/opencode2api/config.json}
listen_address=${LISTEN_ADDRESS:-0.0.0.0:8080}

if [ "$#" -gt 0 ]; then
    exec "$@"
fi

# Docker bind-mounting a non-existent host file creates an empty directory;
# fall back to the example config so the service still starts.
if [ -e "$config_path" ] && [ ! -f "$config_path" ]; then
    rmdir "$config_path" 2>/dev/null || true
fi

if [ ! -f "$config_path" ]; then
    mkdir -p "$(dirname "$config_path")"
    cp /app/config.example.json "$config_path"
    printf '%s\n' "config.json not found; created $config_path from the example. Edit ./config.json and restart."
fi

exec /app/opencode2api -config "$config_path" -listen "$listen_address"