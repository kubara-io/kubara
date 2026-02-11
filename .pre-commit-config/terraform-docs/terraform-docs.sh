#!/usr/bin/env bash

DIR_NAMES=""
CONFIG_FILE_PATH=".pre-commit-config/terraform-docs"
TERRAFORM_DOCS_IMAGE="quay.io/terraform-docs/terraform-docs:0.20.0"
CMD_FOUND=$(which terraform-docs)

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
