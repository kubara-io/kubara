#!/usr/bin/env bash


# pipefail that pipes break
set -euo pipefail

export PATH="$HOME/.local/bin/:$PATH"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MANAGED="${MANAGED:-${PWD}/platform-components/helm}"
CONFIG_FILE="${CONFIG_FILE:-config.yaml}"
CLUSTER_NAME="$(yq -r '.clusters[0].name' "$CONFIG_FILE")"
CONFIGS="${CONFIGS:-platform-configs/${CLUSTER_NAME}/helm}"
OUTPUT_FILE="${OUTPUT_FILE:-}"

[[ -f "$CONFIG_FILE" ]] || { echo "::error::Missing $CONFIG_FILE — run 'kubara generate' first (or cd into its output)"; exit 1; }
[[ -d "$MANAGED" ]]     || { echo "::error::Missing $MANAGED — run 'kubara generate' first"; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "::error::helm not found on PATH"; exit 1; }
command -v yq   >/dev/null 2>&1 || { echo "::error::yq not found on PATH"; exit 1; }

KUBE_VERSION=$(yq -r '.clusters[0].terraform.kubernetesVersion' "$CONFIG_FILE")

PROMETHEUS_STATUS="$(yq -r '.clusters[0].services."kube-prometheus-stack".status // "disabled"' "$CONFIG_FILE")"

# helm template flags advertise the monitoring API only when
# kube-prometheus-stack is enabled, since some charts (eg. traefik) render
# ServiceMonitors guarded by a `fail` on monitoring.coreos.com/v1.
HELM_TEMPLATE_ARGS=(--kube-version "$KUBE_VERSION" --include-crds)
if [[ "$PROMETHEUS_STATUS" == enabled ]]; then
  HELM_TEMPLATE_ARGS+=(--api-versions "monitoring.coreos.com/v1")
fi


echo "Rendering charts from $MANAGED (kube-version=$KUBE_VERSION)" >&2


render_dir="$(mktemp -d)"; trap 'rm -rf "$render_dir"' EXIT
FAILED=()

for chart_path in "$MANAGED"/*/; do
    chart=$(basename "$chart_path")
    [[ -f "$chart_path/Chart.yaml" ]] || continue

    # Don't render library charts
    [[ "$(yq '.type // "application"' "$chart_path/Chart.yaml")" == library ]] && continue

    echo "Updating dependency for ${chart_path}" >&2

    if ! dep_out=$(helm dependency update "$chart_path" >/dev/null 2>&1); then
        echo "::error::helm dependency update failed for '$chart_path'"; echo "$dep_out" >&2
        FAILED+=("$chart:dependency-update"); continue
    fi

    values_file="$CONFIGS/$chart/values.generated.yaml"
    base_values=(); [[ -f "$values_file" ]] && base_values=(-f "$values_file")

    if ! helm template "${HELM_TEMPLATE_ARGS[@]}" \
            "$chart" "$chart_path" "${base_values[@]}" \
            > "$render_dir/$chart.yaml" 2> "$render_dir/$chart.err"; then
        echo "::error::helm template for for '$chart':"
        sed 's/^/    /' "$render_dir/$chart.err" >&2
        FAILED+=("$chart:template"); continue
    fi
done

IMAGES="$(
  cat "$render_dir"/*.yaml |
    grep -E '^[[:space:]]*image:' |
    sed -E "s/^[[:space:]]*image:[[:space:]]*//; s/[\"']//g" |
    grep -vE '[*!]' |          # drop kyverno wildcard/negation entries
    grep -vE '^[[:space:]]*$' |
    sort -u
)"

echo "Done Rendering!"

[[ -n "$IMAGES" ]] || { echo "::warning::No image references found"; exit 0; }

echo "$IMAGES"

if [[ -n "$OUTPUT_FILE" ]]; then
    echo "$IMAGES" > "$OUTPUT_FILE"
    echo "::notice::Image list written to $OUTPUT_FILE"
fi
