#!/bin/sh
set -eu

entrypoint="${1:-docker/entrypoint.sh}"
test_root="$(mktemp -d)"
trap 'rm -rf "${test_root}"' EXIT

sed \
  -e "s#/app/data#${test_root}/data#g" \
  -e "s#/app/log#${test_root}/log#g" \
  -e "s#/app/defaults/geo#${test_root}/geo#g" \
  -e "s#/app/submerge#${test_root}/base#g" \
  "${entrypoint}" > "${test_root}/entrypoint.sh"
chmod +x "${test_root}/entrypoint.sh"

make_probe() {
  path="$1"
  label="$2"
  sed "s#PROBE_LABEL#${label}#g; s#PROBE_LOG#${test_root}/selected#g" <<'EOF' > "${path}"
#!/bin/sh
printf '%s:%s\n' 'PROBE_LABEL' "$*" > 'PROBE_LOG'
EOF
  chmod +x "${path}"
}

make_probe "${test_root}/base" "base"
runtime="${test_root}/runtime"
mkdir -p "${runtime}"

SUBMERGE_UPDATE_DIR="${runtime}" "${test_root}/entrypoint.sh" first
test "$(cat "${test_root}/selected")" = "base:first"

make_probe "${runtime}/submerge" "override"
SUBMERGE_UPDATE_DIR="${runtime}" "${test_root}/entrypoint.sh" second argument
test "$(cat "${test_root}/selected")" = "override:second argument"

printf '%s\n' "entrypoint selection passed"

