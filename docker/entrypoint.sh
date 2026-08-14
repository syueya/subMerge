#!/bin/sh
set -eu

runtime_dir="${SUBMERGE_UPDATE_DIR:-/app/runtime}"

mkdir -p /app/data /app/log /app/defaults/geo "${runtime_dir}"

# Bind mounts commonly arrive owned by root. Drop privileges after making the
# four application-owned directories writable for the configured numeric ID.
if [ "$(id -u)" = "0" ]; then
  run_uid="${PUID:-10001}"
  run_gid="${PGID:-10001}"
  chown -R "${run_uid}:${run_gid}" /app/data /app/log /app/defaults/geo "${runtime_dir}"
  exec su-exec "${run_uid}:${run_gid}" "$0" "$@"
fi

override="${runtime_dir}/submerge"
if [ -f "${override}" ] && [ -x "${override}" ]; then
  exec "${override}" "$@"
fi

exec /app/submerge "$@"

