#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This will canonicalize the path
PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd -P)
BINARY_NAME="config-template"

OUT_DIR=${PROJECT_ROOT}/out

### Go related functions
function go::lint() {
  golangci-lint run "$PROJECT_ROOT"
}

function go::lint-docker() {
  echo "Linting go code in docker container..."
  docker run -t --rm -v "${PROJECT_ROOT}":/app -w /app golangci/golangci-lint:v2.1.6 golangci-lint run
}

function go::build() {
  local OS_TARGETS=${OS_TARGETS:-"linux darwin windows"} ARCH_TARGETS=${ARCH_TARGETS:-"amd64 arm64"}
  for os in $OS_TARGETS; do
    for arch in $ARCH_TARGETS; do
      echo "Building for OS: ${os} with ARCH: ${arch}"
      GOOS=$os GOARCH=$arch go build -o "$OUT_DIR"/"$BINARY_NAME"-"$os"-"$arch" "${PROJECT_ROOT}"
    done
  done
}


### Helm related functions
function helm::dep_update() {
  printf "Running helm dependency update for chart: \n%s\n" "$CHART_PATH"

  local PRIVATE_REPO_URL=${1:-"unspecified"}
  local REPO_USERNAME=${2:-"unspecified"}
  local REPO_PASSWORD=${3:-"unspecified"}

  cd $PROJECT_ROOT

  helm dependency list $CHART_PATH --max-col-width 0 2> /dev/null | tail +2 | sed "\$d" | sed "/^$/d" | \
    grep -v -e 'oci://' | awk -v url="$PRIVATE_REPO_URL" -v user="$REPO_USERNAME" -v passw="$REPO_PASSWORD" '{ if ($3 == url) \
    {print "helm repo add " $1 " " $3 " --username " user " --password " passw } else {print "helm repo add " $1 " " $3} }' | \
    while read cmd; do $cmd && echo "${cmd}" ; done

  cd $CHART_PATH
  helm dependency update
}
function helm::lint() {
  printf "Linting helm chart: \n%s\n" "$CHART_PATH"
  cd $PROJECT_ROOT

  VALUES=""
  for f in "$CHART_PATH"/ci/*\.yaml; do
    [ -e "$f" ] && fname="$(basename "$f")" && VALUES+="--values $CHART_PATH/ci/$fname " && echo "Linting with additional values from: ${fname}" \
    || echo "no ci values found"
  done

  helm lint "$CHART_PATH" --with-subcharts --skip-schema-validation $VALUES
}

function helm::template() {
  printf "Templating helm chart: \n%s\n" "$CHART_PATH"
  cd $PROJECT_ROOT
  VALUES=""
  for f in "$CHART_PATH"/ci/*\.yaml; do
    [ -e "$f" ] && fname="$(basename "$f")" && VALUES+="--values $CHART_PATH/ci/$fname " && echo "Templating with additional values from: ${fname}" \
    || echo "no ci values found"
  done

  helm template $VALUES "$CHART_PATH"
}

function chart::checkVersionBump() {
  local chart="$(basename "$CHART_PATH")"
  echo "Check version bump and changelog for chart: $chart"
  cd $PROJECT_ROOT

  local versionUpdated=1
  local changelogUpdated=1


  # Check if version was updated
  # Regex with grep to only match chart version instead of dependency version
  oldVersion=$(git diff origin -- "$CHART_PATH"/Chart.yaml | grep '^-version:' | awk '{print $2}' || true)
  newVersion=$(git diff origin -- "$CHART_PATH"/Chart.yaml | grep '^+version:' | awk '{print $2}')
  changelogNewVersion=$(git diff origin -- "$CHART_PATH"/CHANGELOG.md | grep '^+## \[' | sed -e "s/.*\[//" -e "s/].*$//" | head -1)

  if [[ $oldVersion == "$newVersion" ]]; then
      versionUpdated=0
  fi
  if [[ $newVersion != "$changelogNewVersion" ]]; then
      echo "##vso[task.logissue type=error]$chart: Version mismatch between Changelog and Chart Version."
      changelogUpdated=0
  fi

  if [[ $changelogNewVersion == '' ]]; then
      changelogUpdated=0
  fi

  if [ $versionUpdated = 0 ] || [ $changelogUpdated = 0 ]; then
      echo "##vso[task.logissue type=error]Error while linting $chart"
      if [ $versionUpdated = 0 ]; then
          echo "##vso[task.logissue type=error]$chart: Chart version not updated"
      fi
      if [ $changelogUpdated = 0 ]; then
          echo "##vso[task.logissue type=error]$chart: Changelog not updated"
      fi
      exit 1
  fi
  echo "##[section]Versioncheck of $chart successful. New Version detected: $newVersion"
}
