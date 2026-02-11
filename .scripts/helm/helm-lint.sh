#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT="$(dirname "${BASH_SOURCE[0]}")/../.."
source "${PROJECT_ROOT}/.scripts/common-functions.sh"

# Path to chart relative to Project Root dir
CHART_PATH="$1"

helm::dep_update
helm::lint
chart::checkVersionBump
