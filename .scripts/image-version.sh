#!/usr/bin/env bash


# pipefail that pipes break
set -euo pipefail

export PATH="$HOME/.local/bin/:$PATH"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MANAGED="${MANAGED:-${PWD}/managed-service-catalog/helm}"
PROFILES="${PROFILES:-${SCRIPT_DIR}/../.github/helm-profiles}"
OUTPUT_FILE="${OUTPUT_FILE:-}"

[[ -d "$MANAGED" ]] || { echo "::errror::MANAGED '$MANAGED' not found - run kubara generate first"; exit 1; }
[[ -d "$PROFILES" ]] || { echo "::error::PROFILES '$PROFILES' not found"; exit 1; }

set -x

render_all() {
    echo "Running"
    for chart_path in "$MANAGED"/*/; do
        [[ -f "$chart_path/Chart.yaml" ]] || continue
        chart=$(basename "$chart_path")

        echo "Updating dependency for ${chart_path}"

        helm dependency update "$chart_path" >/dev/null 2>&1  || continue

        if [[ -d "$PROFILES/$chart" ]]; then
            for profile in "$PROFILES/$chart/*.yaml"; do
                [[ -e "$profile" ]] || continue
                helm template "$chart" "$chart_path" -f "profile" 2>/dev/null
            done
        else
            helm template "$chart" "$chart_path" 2>/dev/null
        fi
    done
}

echo "Rendering templates"

IMAGES=$(
    render_all |
        grep '^\s*image:' |
        sed 's/^\s*image:\s*//' |
        tr -d '"' |
        grep -v '^\s*$' |
        sort -u
)

echo "Done Rendering"

[[ -n "$IMAGES" ]] || { echo "::warning::No image references found"; exit 0; }

echo "$IMAGES"

if [[ -n "$OUTPUT_FILE" ]]; then
    echo "$IMAGES" > "$OUTPUT_FILE"
    echo "::notice::Image list written to $OUTPUT_FILE"
fi
