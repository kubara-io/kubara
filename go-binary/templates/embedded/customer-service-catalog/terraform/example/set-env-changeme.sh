# Create your STACKIT Service Account Key and download your-service-account.json (or use existing)
# https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs#authentication
# move your .json-file to your desired location, set variable accordingly.
export STACKIT_SERVICE_ACCOUNT_KEY_PATH="/path/to/your-service-account.json"

# S3 Bucket credentials (terraform remote state bucket)
# known after "bootstrap-tfstate-backend" or add existing bucket
# AWS_ACCESS_KEY_ID -> Terraform output credential_access_key (from "bootstrap-tfstate-backend" output)
export AWS_ACCESS_KEY_ID=""
# AWS_SECRET_ACCESS_KEY -> Terraform output: credential_secret_access_key (from "bootstrap-tfstate-backend" output)
export AWS_SECRET_ACCESS_KEY=""

# (optional) define grafana admin credentials. Leave as empty string to auto-generate credentials
export TF_VAR_grafana_admin_user=""
export TF_VAR_grafana_admin_password=""

# add known credentials for oAuth2 Apps here
export TF_VAR_image_pull_secret="" #base64 - how to: https://docs.kubara.io/latest-stable/6_reference/faq/#how-do-i-create-a-dockerconfigjson-for-env-file
export TF_VAR_argo_oauth2_client_id=""
export TF_VAR_argo_oauth2_client_secret=""
export TF_VAR_grafana_oauth2_client_id=""
export TF_VAR_grafana_oauth2_client_secret=""
export TF_VAR_oauth2_client_id=""
export TF_VAR_oauth2_client_secret=""



# terraform state rm \
#    vault_kv_secret_v2.image_pull_secret \
#    vault_kv_secret_v2.oauth2_creds \
#    vault_kv_secret_v2.argo_oauth2_creds \
#    vault_kv_secret_v2.grafana_oauth2_creds \
#    random_password.oauth2_cookie_secret
