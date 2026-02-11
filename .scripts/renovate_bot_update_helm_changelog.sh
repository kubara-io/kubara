#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 5 ]; then
  echo "Usage: $0 <depName> <currentVersion> <newVersion> <packageFileDir> <bumpType>"
  exit 1
fi

DEP_NAME=$1
DEP_OLD=$2
DEP_NEW=$3
CHART_DIR=$4
BUMP_TYPE=$5

CHART_FILE="$CHART_DIR/Chart.yaml"
CHANGELOG="$CHART_DIR/CHANGELOG.md"

if [ ! -f "$CHART_FILE" ]; then
  echo "Error: Chart file '$CHART_FILE' not found."
  exit 1
fi

# Function to bump semantic version
bump_version() {
  local ver=$1
  local type=$2
  IFS='.' read -r major minor patch <<<"$ver"
  case "$type" in
    patch)
      patch=$((patch+1))
      ;;
    minor)
      minor=$((minor+1))
      patch=0
      ;;
    major)
      major=$((major+1))
      minor=0
      patch=0
      ;;
    *)
      echo "Unknown bump type: $type" >&2
      exit 1
      ;;
  esac
  echo "$major.$minor.$patch"
}

# Read current chart version (top-level "version:" line)
OLD_CHART_VERSION=$(grep '^version:' "$CHART_FILE" | head -n1 | awk '{print $2}')

# Calculate new chart version from bumpType
NEW_CHART_VERSION=$(bump_version "$OLD_CHART_VERSION" "$BUMP_TYPE")

TODAY=$(date +%F)

# Ensure CHANGELOG exists
if [ ! -f "$CHANGELOG" ]; then
  cat > "$CHANGELOG" <<'EOF'
# Changelog
All notable changes to this chart will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this chart adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF
fi

# Insert new block before first version header
awk -v ver="$NEW_CHART_VERSION" -v date="$TODAY" -v dep="$DEP_NAME" -v old="$DEP_OLD" -v new="$DEP_NEW" '
  BEGIN {
    new_entry = "## [" ver "] - " date "\n### Changed\n- Updated chart dependency version: " dep " " old " → " new "\n"
    inserted = 0
  }
  /^## \[/ && inserted==0 { print new_entry; inserted=1 }
  { print }
  END {
    if (inserted==0) {
      print new_entry
    }
  }
' "$CHANGELOG" > "$CHANGELOG".new && mv "$CHANGELOG".new "$CHANGELOG"
