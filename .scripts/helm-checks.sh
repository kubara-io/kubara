#!/usr/bin/env bash
# Lint + render + validate all managed Helm charts.
#
# The script runs in three sequential phases. Each phase loops over every
# chart under $MANAGED and does exactly one thing:
#
#   PHASE 1 — Render
#     For each chart:
#       a) run `helm dependency update` so sub-charts are available.
#       b) run `helm template` with customer values (the "base" render).
#       c) run `helm template` once per profile overlay from $PROFILES/<chart>.
#       d) extract the openAPIV3Schema from every CustomResourceDefinition
#          into a shared pool under $SCHEMA_POOL. kubeconform ships with
#          schemas for standard Kubernetes resources but not for custom
#          resources (ServiceMonitor, Certificate, ...) — those schemas only
#          exist inside the CustomResourceDefinitions that declare them. The
#          pool is shared across all charts because one chart may define a
#          CustomResourceDefinition that another chart consumes. Two charts
#          defining the same group/kind/version differently are flagged as a
#          collision.
#
#   PHASE 2 — Lint
#     For each chart:
#       a) run `helm lint` with customer values.
#       b) run `helm lint` once per profile overlay.
#
#   PHASE 3 — Validate
#     For each rendered file produced in phase 1:
#       a) strip server-populated fields that would confuse strict validation.
#       b) run `kubeconform` against the schema pool plus kubeconform's
#          built-in Kubernetes schemas.
#
# All failures are collected in FAILED and reported once at the end, so a
# single run surfaces every problem instead of bailing on the first one.
# Precondition: run `kubara generate` first.
#
# Local usage:   ./.scripts/helm-checks.sh

# Bash safety flags:
#   -u          → error if an undefined variable is used
#   -o pipefail → if any command in a pipeline fails, the whole pipeline fails
# We intentionally skip -e: we want the script to continue after a failing
# check so we can collect all failures in FAILED and report them together.
set -uo pipefail

# Tools installed by CI (e.g. kubeconform) live in ~/.local/bin.
# Put that first so we pick them up instead of any older system copies.
export PATH="$HOME/.local/bin:$PATH"

# ─── Paths ────────────────────────────────────────────────────────────
# MANAGED/CUSTOMER are resolved against $PWD because that is where
# `kubara generate` produces its output — CI and local runs both cd there.
# PROFILES is resolved against the script location so the same file layout
# works in both places without symlinks: CI checks out .scripts + .github
# side by side, locally the repo already has them.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MANAGED="managed-service-catalog/helm"
CUSTOMER="customer-service-catalog/helm/kubara"
PROFILES="${PROFILES:-$SCRIPT_DIR/../.github/helm-profiles}"
REPORT_DIR="${REPORT_DIR:-$PWD/reports}"
RENDER_DIR="${RENDER_DIR:-$PWD/rendered}"
SCHEMA_POOL="$RENDER_DIR/schema-pool"

mkdir -p "$REPORT_DIR/helm" "$RENDER_DIR"
rm -rf "$SCHEMA_POOL"
mkdir -p "$SCHEMA_POOL"

# ─── Preconditions ────────────────────────────────────────────────────
[[ -f config.yaml ]] || { echo "::error::Missing config.yaml — run 'kubara generate' first"; exit 1; }
[[ -d $MANAGED ]]    || { echo "::error::Missing $MANAGED — run 'kubara generate' first"; exit 1; }

KUBE_VERSION=$(yq -r '.clusters[] | select(.name=="kubara") | .terraform.kubernetesVersion' config.yaml)
PROMETHEUS_STATUS=$(yq -r '.clusters[] | select(.name=="kubara") | .services.kubePrometheusStack.status // "disabled"' config.yaml)

# helm template flags. We add the monitoring API version only if
# kube-prometheus-stack is enabled, since some charts only render
# ServiceMonitors conditionally on that API being advertised.
# Note: `helm lint` does NOT support --api-versions, only --kube-version.
# Charts with a `fail` guard on monitoring.coreos.com/v1 will emit a harmless
# "funcMap fail" INFO line during lint — lint itself still passes.
HELM_TEMPLATE_ARGS=(--kube-version "$KUBE_VERSION" --include-crds)
if [[ $PROMETHEUS_STATUS == enabled ]]; then
  HELM_TEMPLATE_ARGS+=(--api-versions "monitoring.coreos.com/v1")
  echo "ℹ️ kubePrometheusStack enabled → adding monitoring.coreos.com/v1 API version"
fi

yq --version
kubeconform -v

CHARTS=$(find "$MANAGED" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
TOTAL_CHARTS=$(echo "$CHARTS" | wc -l | tr -d ' ')
FAILED=()

# ════════════════════════════════════════════════════════════════════════
# PHASE 1 — render every chart and collect CustomResourceDefinition schemas
# ════════════════════════════════════════════════════════════════════════
echo ""
echo "══════ PHASE 1: helm template + collect CustomResourceDefinitions ══════"

chart_index=0
for chart in $CHARTS; do
  chart_index=$((chart_index + 1))
  echo "::group::[render $chart_index/$TOTAL_CHARTS] $chart"
  chart_path="$MANAGED/$chart"
  values_file="$CUSTOMER/$chart/values.yaml"
  out_dir="$RENDER_DIR/$chart"
  mkdir -p "$out_dir"

  # Skip non-charts and library charts entirely — they produce no render.
  if [[ ! -f $chart_path/Chart.yaml ]]; then
    echo "::warning::no Chart.yaml — skipping"
    echo "::endgroup::"; continue
  fi
  if [[ $(yq '.type // "application"' "$chart_path/Chart.yaml") == library ]]; then
    echo "::notice::library chart — skipping"
    echo "::endgroup::"; continue
  fi

  # helm dependency update is very chatty ("Hang tight..." × sub-chart). Capture
  # its output and only print it if the command actually failed.
  echo "📦 updating dependencies..."
  if ! dep_output=$(helm dependency update "$chart_path" 2>&1); then
    echo "$dep_output"
    echo "::error file=$chart_path/Chart.yaml::helm dependency update failed"
    FAILED+=("$chart:dependency-update")
    echo "::endgroup::"; continue
  fi

  # base_values = customer values if present, else none.
  # Profiles are only applied on top of customer values.
  base_values=()
  [[ -f $values_file ]] && base_values=(-f "$values_file")

  echo "📄 rendering base → $out_dir/render.yaml"
  helm template "${HELM_TEMPLATE_ARGS[@]}" "$chart" "$chart_path" "${base_values[@]}" \
    > "$out_dir/render.yaml"

  if [[ ${#base_values[@]} -gt 0 && -d $PROFILES/$chart ]]; then
    for profile in "$PROFILES/$chart"/*.yaml; do
      [[ -e $profile ]] || continue
      name=$(basename "$profile" .yaml)
      echo "📄 rendering profile $name → $out_dir/render-$name.yaml"
      helm template "${HELM_TEMPLATE_ARGS[@]}" "$chart-$name" "$chart_path" \
        "${base_values[@]}" -f "$profile" \
        > "$out_dir/render-$name.yaml"
    done
  fi

  # Layout: $SCHEMA_POOL/<group>/<Kind>_<version>.json — kubeconform expects
  # exactly that layout in Phase 3. See the script header for the rationale.
  for render_file in "$out_dir"/render*.yaml; do
    [[ -e $render_file ]] || continue
    # yq produces tab-separated lines (4 fields: group, kind, version, base64 schema).
    # IFS is bash's "Internal Field Separator" — the character that `read`
    # uses to split a line into fields. Default is whitespace, we override
    # it to a single tab so the base64 schema (which may contain spaces)
    # stays in one piece.
    while IFS=$'\t' read -r group kind version schema_b64; do
      [[ -n $group && -n $kind && -n $version && -n $schema_b64 ]] || continue
      schema_file="$SCHEMA_POOL/$group/${kind}_${version}.json"
      mkdir -p "$(dirname "$schema_file")"
      tmp=$(mktemp)
      echo "$schema_b64" | base64 --decode > "$tmp"
      if [[ -f $schema_file ]]; then
        if ! cmp -s "$schema_file" "$tmp"; then
          echo "::error file=$render_file::CRD schema collision for $group/$kind/$version"
          FAILED+=("$chart:schema-collision-$group-$kind-$version")
        fi
        rm -f "$tmp"
      else
        mv "$tmp" "$schema_file"
      fi
    done < <(
      yq -r '
        select(.kind == "CustomResourceDefinition")
        | .spec as $s
        | $s.versions[]
        | select(.schema.openAPIV3Schema != null)
        | [$s.group, $s.names.kind, .name, (.schema.openAPIV3Schema | tojson | @base64)]
        | @tsv
      ' "$render_file"
    )
  done

  echo "::endgroup::"
done

pool_count=$(find "$SCHEMA_POOL" -type f -name '*.json' | wc -l | tr -d ' ')
echo "📦 CustomResourceDefinition schema pool: $pool_count files"

# ════════════════════════════════════════════════════════════════════════
# PHASE 2 — helm lint every chart (base + each profile)
# ════════════════════════════════════════════════════════════════════════
echo ""
echo "══════ PHASE 2: helm lint ══════"

chart_index=0
for chart in $CHARTS; do
  chart_index=$((chart_index + 1))
  # Reuse Phase 1's render.yaml existence as a signal: if it's missing, Phase 1
  # skipped or failed this chart — don't lint it either.
  [[ -f $RENDER_DIR/$chart/render.yaml ]] || continue

  echo "::group::[lint $chart_index/$TOTAL_CHARTS] $chart"
  chart_path="$MANAGED/$chart"
  values_file="$CUSTOMER/$chart/values.yaml"

  base_values=()
  lint_ref="$chart_path/Chart.yaml"
  if [[ -f $values_file ]]; then
    base_values=(-f "$values_file")
    lint_ref="$values_file"
  fi

  echo "🧪 linting base"
  if ! helm lint --quiet --kube-version "$KUBE_VERSION" "$chart_path" "${base_values[@]}" \
      | tee "$REPORT_DIR/helm/$chart-lint.log"; then
    echo "::error file=$lint_ref::helm lint failed"
    FAILED+=("$chart:lint")
  fi

  if [[ ${#base_values[@]} -gt 0 && -d $PROFILES/$chart ]]; then
    for profile in "$PROFILES/$chart"/*.yaml; do
      [[ -e $profile ]] || continue
      name=$(basename "$profile" .yaml)
      echo "🧪 linting profile $name"
      if ! helm lint --quiet --kube-version "$KUBE_VERSION" "$chart_path" "${base_values[@]}" -f "$profile" \
          | tee "$REPORT_DIR/helm/$chart-lint-$name.log"; then
        echo "::error file=$profile::helm lint failed ($name)"
        FAILED+=("$chart:lint:$name")
      fi
    done
  fi

  echo "::endgroup::"
done

# ════════════════════════════════════════════════════════════════════════
# PHASE 3 — kubeconform on every rendered manifest
# ════════════════════════════════════════════════════════════════════════
echo ""
echo "══════ PHASE 3: kubeconform ══════"

TOTAL_RENDERS=$(find "$RENDER_DIR" -type f -name 'render*.yaml' | wc -l | tr -d ' ')
render_index=0
while read -r render_file; do
  render_index=$((render_index + 1))
  chart=$(basename "$(dirname "$render_file")")
  name=$(basename "$render_file" .yaml)

  echo "::group::[validate $render_index/$TOTAL_RENDERS] $chart / $name"
  echo "🔍 validating $render_file"

  # A CustomResourceDefinition's .status field is server-populated; strip it
  # so strict validation doesn't choke on nulls.
  yq eval -i 'select(.kind == "CustomResourceDefinition") |= del(.status)' "$render_file"

  if ! kubeconform \
      -output pretty \
      -strict \
      -kubernetes-version="$KUBE_VERSION" \
      -schema-location "$SCHEMA_POOL/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
      -schema-location default \
      -schema-location "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/{{.NormalizedKubernetesVersion}}/{{.ResourceKind}}.json" \
      "$render_file" \
      | tee "$REPORT_DIR/helm/$chart-$name-kubeconform.log"; then
    echo "::error file=$render_file::kubeconform failed"
    FAILED+=("$chart:$name")
  fi

  echo "::endgroup::"
done < <(find "$RENDER_DIR" -type f -name 'render*.yaml' | sort)

# Drop any log files that ended up empty (no output = no reason to keep them).
find "$REPORT_DIR/helm" -type f -empty -delete

# ════════════════════════════════════════════════════════════════════════
# Summary
# ════════════════════════════════════════════════════════════════════════
echo ""
echo "────────────────────────────────────"
if [[ ${#FAILED[@]} -gt 0 ]]; then
  echo "::error::❌ Failures:"
  for fail in "${FAILED[@]}"; do
    echo "::error::   - $fail"
  done
  exit 1
fi
echo "✅ All Helm checks passed."
