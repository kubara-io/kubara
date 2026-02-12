#!/usr/bin/env bash

DIR_NAMES=""
CONFIG_FILE_PATH=".pre-commit-config/terraform-docs"
TERRAFORM_DOCS_IMAGE="quay.io/terraform-docs/terraform-docs:0.20.0"
# Detect terraform-docs on PATH (local install or injected by CI)
CMD_FOUND="$(command -v terraform-docs || true)"

if [ -n "$CMD_FOUND" ]; then
  echo "[terraform-docs] Using local binary: $CMD_FOUND"
else
  echo "[terraform-docs] Using Docker image: $TERRAFORM_DOCS_IMAGE"
fi

for file in "$@"; do
  DIR_NAMES+="$(dirname "$file") "
done

UNIQ_DIRS=$(echo "${DIR_NAMES}" | tr ' ' '\n' | sort -u)

for dir in $UNIQ_DIRS; do
  if [ "$CMD_FOUND" ]; then
      terraform-docs -c $CONFIG_FILE_PATH/.terraform-docs.yml "$(pwd)/$dir"
  else
    docker run --rm \
      -v "$(pwd)/$dir:/$(basename "$dir")" \
      -v "$(pwd)/$CONFIG_FILE_PATH/.terraform-docs.yml:/terraform-docs-conf/.terraform-docs.yml" \
      -u "$(id -u)" $TERRAFORM_DOCS_IMAGE -c /terraform-docs-conf/.terraform-docs.yml /"$(basename "$dir")"
  fi
done
