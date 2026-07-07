package envconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

type ErrorEnvMap struct {
	Message string
	Err     error
}

var ErrEnvsNotSet = errors.New("EnvVars have not been set")
var ErrDefaultIsSet = errors.New("EnvVars are set to default value")

func (e *ErrorEnvMap) Error() string {
	return fmt.Sprintf("Error: %s", e.Message)
}

func (e *ErrorEnvMap) Unwrap() error {
	return e.Err
}

// EnvMap holds the expected variables
type EnvMap struct {
	_                           struct{} `doc:"# ✅ These values MUST be known BEFORE running Terraform."`
	_                           struct{} `doc:"# 🔁 Everything in <angle brackets> MUST be replaced."`
	_                           struct{} `doc:"# 💡 Values without <> are optional and can be left as-is if not needed (e.g. no private image registry)."`
	_                           struct{} `doc:"#    It will still create a secret, but it won't be valid."`
	_                           struct{} `doc:"\n# Project related values"`
	ProjectName                 string   `default:"<...>" koanf:"PROJECT_NAME"`
	ProjectStage                string   `default:"<...>" koanf:"PROJECT_STAGE"`
	_                           struct{} `doc:"\n# DNS Name/Zones related values"`
	_                           struct{} `doc:"# The Domain name under which your dns-entries will be added."`
	_                           struct{} `doc:"# The resulting dnsName name will be a concatenation of <PROJECT_NAME>.<DOMAIN_NAME> and will be used in clusters[].dnsName and clusters[].terraform.dns.name in config.yaml"`
	_                           struct{} `doc:"# the value should be looking like 'stackit.zone' eg. 'yourDomain.com'"`
	DomainName                  string   `default:"<...>" koanf:"DOMAIN_NAME"`
	_                           struct{} `doc:"\n# Argo CD related values"`
	_                           struct{} `doc:"# Initial Admin Account Password for Argo CD."`
	ArgocdWizardAccountPassword string   `default:"<...>" koanf:"ARGOCD_WIZARD_ACCOUNT_PASSWORD"`
	_                           struct{} `doc:"\n# Git repository values"`
	_                           struct{} `doc:"# The HTTPS URL of the git repository that Argo CD will use to pull the kubara generated manifests from."`
	ArgocdGitHttpsUrl           string   `default:"<...>" koanf:"ARGOCD_GIT_HTTPS_URL"`
	_                           struct{} `doc:"\n# Optional: Git repository credentials for Argo to pull your kubara manifests from."`
	_                           struct{} `doc:"# Necessary if your repository isn't public."`
	ArgocdGitUsername           string   `default:"" koanf:"ARGOCD_GIT_USERNAME" optional:"true"`
	ArgocdGitPatOrPassword      string   `default:"" koanf:"ARGOCD_GIT_PAT_OR_PASSWORD" optional:"true"`
	_                           struct{} `doc:"\n# Optional: Helm repository values (leave empty to disable)."`
	_                           struct{} `doc:"# ARGOCD_HELM_REPO_URL supports: https://... (classic Helm repo) or registry.example.com/... (OCI Helm registry)."`
	_                           struct{} `doc:"# Compatibility: oci://... is also accepted and normalized automatically."`
	ArgocdHelmRepoUsername      string   `default:"" koanf:"ARGOCD_HELM_REPO_USERNAME" optional:"true"`
	ArgocdHelmRepoPassword      string   `default:"" koanf:"ARGOCD_HELM_REPO_PASSWORD" optional:"true"`
	ArgocdHelmRepoUrl           string   `default:"" koanf:"ARGOCD_HELM_REPO_URL" optional:"true"`
	_                           struct{} `doc:"\n# Optional: Container Registry Config"`
	_                           struct{} `doc:"# the variable must be base64 encoded - how to: https://docs.kubara.io/latest-stable/6_reference/faq/#how-do-i-create-a-dockerconfigjson-for-env-file"`
	DockerconfigBase64          string   `default:"" koanf:"DOCKERCONFIG_BASE64" optional:"true"`
}

// Validate performs basic validation on the envMap.
// It looks at all fields but only raises an error if non optional fields are not set or set to default.
func (em *EnvMap) Validate() error {
	v := reflect.ValueOf(em).Elem()
	t := v.Type()

	var varsNotSet, defaultIsSet []string
	var emptyVarsE, defaultIsSetE *ErrorEnvMap

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Tag.Get("koanf")
		defaultTagVal := fieldType.Tag.Get("default")
		isOptional := fieldType.Tag.Get("optional") == "true"

		if utils.IsZeroValue(field) {
			if !isOptional {
				varsNotSet = append(varsNotSet, fieldName)
			}
		}
		if utils.IsDefaultValue(field, defaultTagVal) && !isOptional {
			defaultIsSet = append(defaultIsSet, fieldName)
		}
	}

	if len(varsNotSet) > 0 {
		errText := fmt.Sprintf("Vars not set: %+v", varsNotSet)
		emptyVarsE = &ErrorEnvMap{
			Message: errText,
			Err:     ErrEnvsNotSet,
		}
		return emptyVarsE
	}
	if len(defaultIsSet) > 0 {
		errText := fmt.Sprintf("Vars are set to default: %+v", defaultIsSet)
		defaultIsSetE = &ErrorEnvMap{
			Message: errText,
			Err:     ErrDefaultIsSet,
		}
		return defaultIsSetE
	}

	return nil
}

// setDefaults sets default values for empty fields based on the struct tag "default"
func (em *EnvMap) setDefaults() {
	v := reflect.ValueOf(em).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		defaultTagValue := fieldType.Tag.Get("default")

		if utils.IsZeroValue(field) {
			if defaultTagValue != "" {
				utils.SetFieldValue(field, defaultTagValue)
			}
		}
	}
}

// IsConfiguredEnvValue reports whether a value is explicitly configured by the user.
// Empty strings and legacy "<...>" placeholders are treated as not configured.
func IsConfiguredEnvValue(v string) bool {
	trimmed := strings.TrimSpace(v)
	return trimmed != "" && trimmed != "<...>"
}

// NormalizeHelmRepoURL normalizes Helm repository inputs for ArgoCD.
// If oci:// is provided, it is removed because ArgoCD helm repository
// credentials expect the registry URL without the scheme.
func NormalizeHelmRepoURL(v string) string {
	trimmed := strings.TrimSpace(v)
	if strings.HasPrefix(strings.ToLower(trimmed), "oci://") {
		return trimmed[len("oci://"):]
	}
	return trimmed
}

// IsOCIHelmRepoURL reports whether a Helm repository URL should be treated
// as OCI. HTTPS/HTTP URLs are treated as classic Helm repos.
func IsOCIHelmRepoURL(v string) bool {
	normalized := NormalizeHelmRepoURL(v)
	if normalized == "" {
		return false
	}
	lower := strings.ToLower(normalized)
	return !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://")
}
