#!/usr/bin/env bash
set -euo pipefail

# Keep Terraform validation logic in a dedicated script for easier local replay
# and workflow readability.
# Precondition: generated output must exist (run `kubara generate` first).
#
# Local usage (from repo root):
#   ./.scripts/terraform-checks.sh

# ---- Runtime Defaults -------------------------------------------------------
# CI provides REPORT_DIR. For local runs we use ./reports by default.
if [[ -z "${REPORT_DIR:-}" ]]; then
  REPORT_DIR="${PWD}/reports"
  echo "ℹ️ REPORT_DIR not set, using default: $REPORT_DIR"
fi

# Keep per-directory logs under REPORT_DIR to make failures easy to inspect.
mkdir -p "${REPORT_DIR}/terraform"

echo ""
echo "📦 Starting Terraform static analysis"
echo "──────────────────────────────────────────────"
# Avoid piping to `head` here: with `set -euo pipefail`, an upstream SIGPIPE
# can abort the script (seen with some tool/version combinations in CI).
terraform version
tflint --version

FAILED_DIRS=()
TARGET_DIRS=(
  "managed-service-catalog/terraform/modules"
  "customer-service-catalog/terraform/kubara"
)

# ---- Preconditions ----------------------------------------------------------
# This script validates generated Terraform output and expects kubara generate
# to have produced the target directories first.
HAS_GENERATED_TERRAFORM_DIRS=false
for BASE_DIR in "${TARGET_DIRS[@]}"; do
  if [[ -d "$BASE_DIR" ]]; then
    HAS_GENERATED_TERRAFORM_DIRS=true
    break
  fi
done

if [[ "$HAS_GENERATED_TERRAFORM_DIRS" != true ]]; then
  echo "::error::No generated Terraform directories found. Run 'kubara generate' first."
  exit 1
fi

# Validate each generated Terraform directory with fmt/init/validate/tflint.
# We continue collecting failures and fail once at the end with a full summary.
for BASE_DIR in "${TARGET_DIRS[@]}"; do
  [[ -d "$BASE_DIR" ]] || continue

  for DIR in "$BASE_DIR"/*; do
    [[ -d "$DIR" ]] || continue

    NAME="$(basename "$(dirname "$DIR")")-$(basename "$DIR")"
    LOG="${REPORT_DIR}/terraform/${NAME}.log"

    echo "────────────────────────────────────────────"
    echo "🔍 Checking: $DIR"
    echo "📝 Report: $LOG"
    echo "────────────────────────────────────────────"
    echo "::group::Terraform check: $NAME"

    echo "📂 Directory: $DIR" | tee "$LOG"

    echo "📐 terraform fmt -check -diff" | tee -a "$LOG"
    if ! terraform fmt -check -diff "$DIR" | tee -a "$LOG"; then
      echo "::error file=$DIR::terraform fmt failed"
      FAILED_DIRS+=("$DIR")
    fi

    echo "📋 terraform init (backend=false)" | tee -a "$LOG"
    if ! terraform -chdir="$DIR" init -backend=false -input=false >> "$LOG" 2>&1; then
      echo "::error file=$DIR::terraform init failed"
      FAILED_DIRS+=("$DIR")
    fi

    echo "🧪 terraform validate" | tee -a "$LOG"
    if ! terraform -chdir="$DIR" validate | tee -a "$LOG"; then
      echo "::error file=$DIR::terraform validate failed"
      FAILED_DIRS+=("$DIR")
    fi

    echo "🔎 tflint" | tee -a "$LOG"
    if ! tflint --chdir "$DIR" | tee -a "$LOG"; then
      echo "::error file=$DIR::tflint failed"
      FAILED_DIRS+=("$DIR")
    fi

    echo "✅ Done: $DIR" | tee -a "$LOG"
    echo "::endgroup::"
  done
done

echo ""
echo "──────────────────────────────────────────────"
if [[ "${#FAILED_DIRS[@]}" -gt 0 ]]; then
  echo "::error::❌ The following Terraform directories failed:"
  for FAIL in "${FAILED_DIRS[@]}"; do
    echo "::error::$FAIL"
  done
  exit 1
fi

echo "✅ All Terraform checks passed."
