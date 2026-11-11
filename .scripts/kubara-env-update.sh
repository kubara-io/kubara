#!/usr/bin/env bash
# Update an existing .env file (e.g. copied from env.example).
# Strict template mode: only replace existing keys, never append new ones.

set -euo pipefail

die() { printf "✖ %s\n" "$*" >&2; exit 1; }
log() { printf "▶ %s\n" "$*"; }

command -v sed  >/dev/null 2>&1 || die "Missing command: sed"
command -v grep >/dev/null 2>&1 || die "Missing command: grep"

ENV_FILE="${1:-}"
[[ -n "$ENV_FILE" ]] || die "Usage: .scripts/kubara-env-update.sh <path-to-.env>"
[[ -f "$ENV_FILE" ]] || die ".env not found: $ENV_FILE"

# If $KUBARA_* is set, replace KEY=... with KEY="value" in the existing template.
# Fails if KEY does not exist in the template.
apply_if_set() {
  local src_env="$1"  # e.g. KUBARA_PROJECT_NAME
  local dst_key="$2"  # e.g. PROJECT_NAME

  local val="${!src_env:-}"
  [[ -n "$val" ]] || return 0

  grep -qE "^${dst_key}=" "$ENV_FILE" \
    || die "Template key missing in .env: ${dst_key} (because ${src_env} is set)"

  # Escape special characters before passing to sed:
  # 1. Escape backslashes first so later escapes are not doubled
  local escaped="$val"
  escaped="${escaped//\\/\\\\}"

  # 2. Convert literal newlines and carriage returns into \n / \r sequences
  #    so sed receives a single command line without premature termination ("unterminated 's' command")
  escaped="${escaped//$'\n'/\\n}"
  escaped="${escaped//$'\r'/\\r}"

  # 3. Escape double quotes for the double-quoted value assignment KEY="..."
  escaped="${escaped//\"/\\\"}"

  # 4. Escape '&' (which sed interprets as the matched text reference in replacements)
  escaped="${escaped//&/\\&}"

  # 5. Escape the sed pattern delimiter '|'
  escaped="${escaped//|/\\|}"

  sed -i -e "s|^${dst_key}=.*$|${dst_key}=\"${escaped}\"|g" "$ENV_FILE"
}

log "Updating .env (strict mode, no secret output): $ENV_FILE"

# --- KUBARA_* -> .env KEY ---
# Platform credentials
apply_if_set KUBARA_DOCKERCONFIG_BASE64    DOCKERCONFIG_BASE64

# Git repository credentials
apply_if_set KUBARA_GITOPS_USERNAME                       GITOPS_USERNAME
apply_if_set KUBARA_GITOPS_PASSWORD                       GITOPS_PASSWORD
apply_if_set KUBARA_GITOPS_SSH_PRIVATE_KEY                GITOPS_SSH_PRIVATE_KEY
apply_if_set KUBARA_GITOPS_SSH_PRIVATE_KEY_PATH           GITOPS_SSH_PRIVATE_KEY_PATH
apply_if_set KUBARA_GITOPS_GITHUB_APP_ID                  GITOPS_GITHUB_APP_ID
apply_if_set KUBARA_GITOPS_GITHUB_APP_INSTALLATION_ID     GITOPS_GITHUB_APP_INSTALLATION_ID
apply_if_set KUBARA_GITOPS_GITHUB_APP_PRIVATE_KEY         GITOPS_GITHUB_APP_PRIVATE_KEY
apply_if_set KUBARA_GITOPS_GITHUB_APP_PRIVATE_KEY_PATH    GITOPS_GITHUB_APP_PRIVATE_KEY_PATH
apply_if_set KUBARA_GITOPS_GITHUB_APP_ENTERPRISE_BASE_URL GITOPS_GITHUB_APP_ENTERPRISE_BASE_URL

# Helm repository credentials
apply_if_set KUBARA_GITOPS_HELM_REPO_USERNAME             GITOPS_HELM_REPO_USERNAME
apply_if_set KUBARA_GITOPS_HELM_REPO_PASSWORD             GITOPS_HELM_REPO_PASSWORD

# Legacy ARGOCD_* aliases
apply_if_set KUBARA_ARGOCD_GIT_USERNAME                   GITOPS_USERNAME
apply_if_set KUBARA_ARGOCD_GIT_PAT_OR_PASSWORD            GITOPS_PASSWORD
apply_if_set KUBARA_ARGOCD_GIT_SSH_PRIVATE_KEY            GITOPS_SSH_PRIVATE_KEY
apply_if_set KUBARA_ARGOCD_GIT_GITHUB_APP_ID              GITOPS_GITHUB_APP_ID
apply_if_set KUBARA_ARGOCD_GIT_GITHUB_APP_INSTALLATION_ID GITOPS_GITHUB_APP_INSTALLATION_ID
apply_if_set KUBARA_ARGOCD_GIT_GITHUB_APP_PRIVATE_KEY     GITOPS_GITHUB_APP_PRIVATE_KEY
apply_if_set KUBARA_ARGOCD_GIT_GITHUB_APP_ENTERPRISE_BASE_URL GITOPS_GITHUB_APP_ENTERPRISE_BASE_URL
apply_if_set KUBARA_ARGOCD_HELM_REPO_USERNAME             GITOPS_HELM_REPO_USERNAME
apply_if_set KUBARA_ARGOCD_HELM_REPO_PASSWORD             GITOPS_HELM_REPO_PASSWORD
# ----------------------------

log "✅ .env updated"
