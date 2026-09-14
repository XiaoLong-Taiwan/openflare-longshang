#!/bin/sh
set -eu

RUNTIME_USER="openflare"
AGENT_BIN="/usr/local/bin/openflare-agent"

fix_runtime_ownership() {
  for target in /data /etc/openflare; do
    if [ -d "$target" ]; then
      chown -R "${RUNTIME_USER}:${RUNTIME_USER}" "$target" 2>/dev/null || true
      chmod -R u+rwX,g+rX "$target" 2>/dev/null || true
    fi
  done
  # Allow the runtime user to write to /usr/local/bin to support container self-updating
  chown "${RUNTIME_USER}:${RUNTIME_USER}" /usr/local/bin "${AGENT_BIN}" 2>/dev/null || true
}

if [ "$(id -u)" -eq 0 ]; then
  fix_runtime_ownership
  exec su-exec "${RUNTIME_USER}" "${AGENT_BIN}" "$@"
fi

exec "${AGENT_BIN}" "$@"