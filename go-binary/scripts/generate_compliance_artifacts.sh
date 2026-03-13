#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
GO_BINARY_DIR="$(cd -- "${SCRIPT_DIR}/.." >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(cd -- "${GO_BINARY_DIR}/.." >/dev/null 2>&1 && pwd)"

OUTPUT_DIR="${1:-${GO_BINARY_DIR}/compliance}"
if [[ "${OUTPUT_DIR}" != /* ]]; then
  OUTPUT_DIR="${GO_BINARY_DIR}/${OUTPUT_DIR}"
fi

INCLUDE_ROOT_FILES="${INCLUDE_ROOT_FILES:-true}"
GO_LICENSES_BIN="${GO_LICENSES_BIN:-go-licenses}"

if [[ "${GO_LICENSES_BIN}" == */* ]]; then
  if [[ ! -x "${GO_LICENSES_BIN}" ]]; then
    echo "go-licenses binary not found at: ${GO_LICENSES_BIN}" >&2
    exit 1
  fi
else
  if ! command -v "${GO_LICENSES_BIN}" >/dev/null 2>&1; then
    echo "go-licenses binary not found in PATH: ${GO_LICENSES_BIN}" >&2
    exit 1
  fi
fi

mkdir -p "${OUTPUT_DIR}"
rm -rf "${OUTPUT_DIR}/licenses"

pushd "${GO_BINARY_DIR}" >/dev/null
"${GO_LICENSES_BIN}" report ./... | sort -u > "${OUTPUT_DIR}/THIRD_PARTY_GO_LICENSES.csv"
"${GO_LICENSES_BIN}" save ./... --save_path "${OUTPUT_DIR}/licenses"
popd >/dev/null

if [[ "${INCLUDE_ROOT_FILES}" == "true" ]]; then
  cp "${REPO_ROOT}/LICENSE" "${OUTPUT_DIR}/LICENSE"
  cp "${REPO_ROOT}/NOTICE.txt" "${OUTPUT_DIR}/NOTICE.txt"

  if [[ -f "${REPO_ROOT}/THIRD_PARTY_HELM_DEPENDENCIES.md" ]]; then
    cp "${REPO_ROOT}/THIRD_PARTY_HELM_DEPENDENCIES.md" "${OUTPUT_DIR}/THIRD_PARTY_HELM_DEPENDENCIES.md"
  fi
fi

echo "Compliance artifacts written to ${OUTPUT_DIR}"
