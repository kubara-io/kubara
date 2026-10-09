#!/usr/bin/env bash
# Exercise the real YAML updater with local stand-ins for registry and CLI calls.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/work"
export CALL_LOG="$tmp/calls"
cat > "$tmp/bin/oras" <<'STUB'
#!/usr/bin/env bash
set -eu
[ "${FAIL_TAGS:-false}" = false ] || exit 1
printf '%s\n' latest 1.9.0 1.10.0 2.0.0-rc.1 99x0x0
STUB
cat > "$tmp/bin/kubara" <<'STUB'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >> "$CALL_LOG"
if [ "$1" = catalog ]; then
    [ "${FAIL_PULL:-false}" = false ]
else
    [ "${FAIL_GENERATE:-false}" = false ]
fi
STUB
chmod +x "$tmp/bin/"*
export PATH="$tmp/bin:$PATH"
cat > "$tmp/work/config.yaml" <<'YAML'
# Preserve this comment.
bootstrapCatalog: oci://ghcr.io/kubara-io/catalogs/bootstrap:1.0.0
clusters:
  - name: first
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:1.2.0
      - ./custom
    description: oci://ghcr.io/kubara-io/catalogs/general:1.2.0
  - name: second
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:3.0.0
YAML
cp "$tmp/work/config.yaml" "$tmp/original.yaml"
update() { bash "$root/src/cmd/kubara-catalog-update.sh" "$tmp/work"; }
update
[ "$(yq '.bootstrapCatalog' "$tmp/work/config.yaml")" = oci://ghcr.io/kubara-io/catalogs/bootstrap:1.10.0 ]
[ "$(yq '.clusters[0].catalogs[0]' "$tmp/work/config.yaml")" = oci://ghcr.io/kubara-io/catalogs/general:1.10.0 ]
[ "$(yq '.clusters[1].catalogs[0]' "$tmp/work/config.yaml")" = oci://ghcr.io/kubara-io/catalogs/general:3.0.0 ]
[ "$(yq '.clusters[0].catalogs[1]' "$tmp/work/config.yaml")" = ./custom ]
[ "$(yq '.clusters[0].description' "$tmp/work/config.yaml")" = oci://ghcr.io/kubara-io/catalogs/general:1.2.0 ]
grep -q '^# Preserve this comment.' "$tmp/work/config.yaml"
cp "$tmp/work/config.yaml" "$tmp/updated.yaml"
update
cmp "$tmp/updated.yaml" "$tmp/work/config.yaml"

for failure in FAIL_TAGS FAIL_PULL FAIL_GENERATE; do
    cp "$tmp/original.yaml" "$tmp/work/config.yaml"
    : > "$CALL_LOG"
    if env "$failure=true" bash "$root/src/cmd/kubara-catalog-update.sh" "$tmp/work"; then
        echo "Expected failure: $failure" >&2
        exit 1
    fi
    cmp "$tmp/original.yaml" "$tmp/work/config.yaml"
    if [ "$failure" != FAIL_GENERATE ]; then
        if grep -q generate "$CALL_LOG"; then
            echo "Generation ran after an earlier failure" >&2
            exit 1
        fi
    fi
done

yq -i 'del(.bootstrapCatalog)' "$tmp/work/config.yaml"
: > "$CALL_LOG"
if update; then
    echo 'Expected missing bootstrap pin to fail' >&2
    exit 1
fi
[ ! -s "$CALL_LOG" ]
# Exercise the composite action's literal argv handling, including empty arguments.
yq -r '.runs.steps[1].run' "$root/action.yml" > "$tmp/action-run.sh"
cat > "$tmp/bin/kubara" <<'STUB'
#!/bin/sh
printf '%s\0' "$@" > "$CALL_LOG"
STUB
export KUBARA_WORK_DIR="$tmp/work"
# shellcheck disable=SC2016 # This must reach the CLI literally.
export KUBARA_ARGS='["generate", "a b", "", "$(exit 99)"]'
bash -e -o pipefail "$tmp/action-run.sh"
# shellcheck disable=SC2016 # Expected literal command substitution text.
printf '%s\0' generate 'a b' '' '$(exit 99)' > "$tmp/expected-args"
cmp "$tmp/expected-args" "$CALL_LOG"
if KUBARA_ARGS='[1]' bash -e -o pipefail "$tmp/action-run.sh"; then
    echo 'Expected invalid arguments to fail' >&2
    exit 1
fi
: > "$CALL_LOG"
KUBARA_ARGS='[]' bash -e -o pipefail "$tmp/action-run.sh"
[ ! -s "$CALL_LOG" ]
# Run the GitLab publication block with no network or real git mutations.
yq -r '."kubara-update".script[-1]' "$root/src/cmd/gitlab-catalog-update.yaml" > "$tmp/publish.sh"
cat > "$tmp/bin/git" <<'STUB'
#!/usr/bin/env bash
set -eu
case "$1" in
    diff) [ "$TEST_MODE" = no-change ] ;;
    ls-remote) printf 'abc\trefs/heads/automation/kubara-catalog-update\n' ;;
    write-tree) echo new-tree ;;
    rev-parse) if [ "$TEST_MODE" = identical ]; then echo new-tree; else echo old-tree; fi ;;
    -c)
        if [[ "$*" == *' push '* ]]; then
            echo PUSH >> "$CALL_LOG"
            [ "$TEST_MODE" != push-failure ]
        fi
        ;;
esac
STUB
cat > "$tmp/bin/curl" <<'STUB'
#!/usr/bin/env bash
set -eu
if [[ "$*" == *'--get'* ]]; then
    echo GET >> "$CALL_LOG"
    if [ "$TEST_MODE" = identical ]; then
        echo '[{"web_url":"https://gitlab.test/mr/1"}]'
    else
        echo '[]'
    fi
else
    echo POST >> "$CALL_LOG"
    [ "$TEST_MODE" != api-failure ] || exit 22
    echo '{"web_url":"https://gitlab.test/mr/1"}'
fi
STUB
chmod +x "$tmp/bin/git" "$tmp/bin/curl"
export CI_API_V4_URL=https://gitlab.test/api/v4 CI_PROJECT_ID=1 CI_DEFAULT_BRANCH=main
export CI_PROJECT_URL=https://gitlab.test/example KUBARA_UPDATE_TOKEN=fixture
for mode in no-change identical push-failure api-failure create; do
    : > "$CALL_LOG"
    if (cd "$tmp/work" && TEST_MODE="$mode" sh "$tmp/publish.sh"); then
        [ "$mode" != push-failure ] && [ "$mode" != api-failure ]
    else
        [ "$mode" = push-failure ] || [ "$mode" = api-failure ]
    fi
    if [ "$mode" = create ] || [ "$mode" = api-failure ]; then
        [ "$(grep -c '^POST$' "$CALL_LOG")" -eq 1 ]
        [ "$(grep -c '^PUSH$' "$CALL_LOG")" -eq 1 ]
    elif grep -q '^POST$' "$CALL_LOG"; then
        echo "Unexpected MR creation for $mode" >&2
        exit 1
    fi
done
echo 'Catalog automation tests passed'
