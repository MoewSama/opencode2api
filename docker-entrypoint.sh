#!/bin/sh
set -eu

state_dir=${STATE_DIR:-/var/lib/opencode2api}
config_path=${CONFIG_PATH:-$state_dir/config.json}
config_seed_path=${CONFIG_SEED_PATH:-}
listen_address=${LISTEN_ADDRESS:-0.0.0.0:8080}

if [ "$#" -gt 0 ]; then
    exec "$@"
fi

mkdir -p "$(dirname "$config_path")"
if [ -n "$config_seed_path" ] && [ -f "$config_seed_path" ]; then
    cp "$config_seed_path" "$config_path"
    printf '%s\n' "Loaded $config_path from $config_seed_path."
elif [ ! -f "$config_path" ]; then
    cp /app/config.example.json "$config_path"
    printf '%s\n' \
        "config.json not found; created $config_path. Set API keys or enable anonymous mode, and change the WebUI password before use."
fi

exec /app/opencode2api \
    -config "$config_path" \
    -listen "$listen_address"
