package envconfig

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

type ErrorEnvMap struct {
	Message string
	Err     error
}

var ErrEnvsNotSet = errors.New("EnvVars have not been set")
var ErrDefaultIsSet = errors.New("EnvVars are set to default value")
var ErrInvalidEnvValue = errors.New("EnvVars contain invalid value")

const (
	GitAuthModeHTTPS     = "https"
	GitAuthModeSSH       = "ssh"
	GitAuthModeGitHubApp = "github-app"
)

func (e *ErrorEnvMap) Error() string {
	return fmt.Sprintf("Error: %s", e.Message)
}

func (e *ErrorEnvMap) Unwrap() error {
	return e.Err
}

// EnvMap holds the expected variables
type EnvMap struct {
	_                                   struct{} `doc:"# ✅ These values MUST be known BEFORE running Terraform."`
	_                                   struct{} `doc:"# 🔁 Everything in <angle brackets> MUST be replaced."`
	_                                   struct{} `doc:"# 💡 Dummy values (without <>) are optional and can be left as-is if not needed"`
	_                                   struct{} `doc:"#    (e.g. no private image registry). It will still create a secret, but it will be not valid."`
	_                                   struct{} `doc:"\n### Project related values"`
	ProjectName                         string   `default:"<...>" koanf:"PROJECT_NAME"`
	ProjectStage                        string   `default:"<...>" koanf:"PROJECT_STAGE"`
	_                                   struct{} `doc:"\n### Container Registry Config"`
	_                                   struct{} `doc:"# the variable must be base64 encoded - how to: https://docs.kubara.io/latest-stable/6_reference/faq/#how-do-i-create-a-dockerconfigjson-for-env-file"`
	DockerconfigBase64                  string   `default:"<...>" koanf:"DOCKERCONFIG_BASE64"`
	_                                   struct{} `doc:"\n### Argo CD related values"`
	ArgocdWizardAccountPassword         string   `default:"<...>" koanf:"ARGOCD_WIZARD_ACCOUNT_PASSWORD"`
	_                                   struct{} `doc:"\n### Git repository values"`
	_                                   struct{} `doc:"# ARGOCD_GIT_AUTH_MODE supports: https, ssh, github-app. Empty keeps the legacy https mode."`
	ArgocdGitAuthMode                   string   `default:"https" koanf:"ARGOCD_GIT_AUTH_MODE" optional:"true"`
	_                                   struct{} `doc:"# Prefer ARGOCD_GIT_URL for new setups. ARGOCD_GIT_HTTPS_URL is kept for backward compatibility with existing .env files."`
	ArgocdGitUrl                        string   `default:"" koanf:"ARGOCD_GIT_URL" optional:"true"`
	ArgocdGitHttpsUrl                   string   `default:"<...>" koanf:"ARGOCD_GIT_HTTPS_URL" optional:"true"`
	_                                   struct{} `doc:"# HTTPS mode uses username + password/PAT. PAT usually means Personal Access Token; prefer a technical or machine account, not a personal user account."`
	ArgocdGitPatOrPassword              string   `default:"<...>" koanf:"ARGOCD_GIT_PAT_OR_PASSWORD" optional:"true"`
	ArgocdGitUsername                   string   `default:"<...>" koanf:"ARGOCD_GIT_USERNAME" optional:"true"`
	_                                   struct{} `doc:"# SSH mode uses ARGOCD_GIT_SSH_PRIVATE_KEY and requires trusted SSH host keys in Argo CD known_hosts."`
	ArgocdGitSshPrivateKey              string   `default:"" koanf:"ARGOCD_GIT_SSH_PRIVATE_KEY" optional:"true"`
	_                                   struct{} `doc:"# GitHub App mode uses the GitHub App IDs and private key. Enterprise base URL is optional."`
	ArgocdGitGithubAppID                string   `default:"" koanf:"ARGOCD_GIT_GITHUB_APP_ID" optional:"true"`
	ArgocdGitGithubAppInstallationID    string   `default:"" koanf:"ARGOCD_GIT_GITHUB_APP_INSTALLATION_ID" optional:"true"`
	ArgocdGitGithubAppPrivateKey        string   `default:"" koanf:"ARGOCD_GIT_GITHUB_APP_PRIVATE_KEY" optional:"true"`
	ArgocdGitGithubAppEnterpriseBaseUrl string   `default:"" koanf:"ARGOCD_GIT_GITHUB_APP_ENTERPRISE_BASE_URL" optional:"true"`
	_                                   struct{} `doc:"\n### DNS Name/Zones related values"`
	_                                   struct{} `doc:"# The Domain name under which your dns-entries will be added."`
	_                                   struct{} `doc:"# The resulting dnsZone name will be a concatenation of <PROJECT_NAME>-<PROJECT_STAGE>.<DOMAIN_NAME>"`
	_                                   struct{} `doc:"# the value should be looking like 'stackit.zone' eg. 'yourDomain.com'"`
	DomainName                          string   `default:"<...>" koanf:"DOMAIN_NAME"`
	_                                   struct{} `doc:"\n### Optional values"`
	_                                   struct{} `doc:"# Helm repository values (leave empty to disable)."`
	_                                   struct{} `doc:"# ARGOCD_HELM_REPO_URL supports: https://... (classic Helm repo) or registry.example.com/... (OCI Helm registry)."`
	_                                   struct{} `doc:"# Compatibility: oci://... is also accepted and normalized automatically."`
	ArgocdHelmRepoUsername              string   `default:"" koanf:"ARGOCD_HELM_REPO_USERNAME" optional:"true"`
	ArgocdHelmRepoPassword              string   `default:"" koanf:"ARGOCD_HELM_REPO_PASSWORD" optional:"true"`
	ArgocdHelmRepoUrl                   string   `default:"" koanf:"ARGOCD_HELM_REPO_URL" optional:"true"`
}

// ValidateAll performs basic validation on the envMap.
func (em *EnvMap) ValidateAll() error {
	if err := em.Validate(); err != nil {
		return err
	}
	return nil
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

	if err := em.validateGitAuth(); err != nil {
		return err
	}

	return nil
}

func (em *EnvMap) validateGitAuth() error {
	switch em.GitAuthMode() {
	case GitAuthModeHTTPS:
		return validateRequiredEnvValues(map[string]string{
			"ARGOCD_GIT_URL or ARGOCD_GIT_HTTPS_URL": em.GitRepositoryURL(),
			"ARGOCD_GIT_USERNAME":                    em.ArgocdGitUsername,
			"ARGOCD_GIT_PAT_OR_PASSWORD":             em.ArgocdGitPatOrPassword,
		})
	case GitAuthModeSSH:
		if err := validateRequiredEnvValues(map[string]string{
			"ARGOCD_GIT_URL":             em.ArgocdGitUrl,
			"ARGOCD_GIT_SSH_PRIVATE_KEY": em.ArgocdGitSshPrivateKey,
		}); err != nil {
			return err
		}
		return validateSSHGitURL(em.ArgocdGitUrl)
	case GitAuthModeGitHubApp:
		if err := validateRequiredEnvValues(map[string]string{
			"ARGOCD_GIT_URL":                        em.ArgocdGitUrl,
			"ARGOCD_GIT_GITHUB_APP_ID":              em.ArgocdGitGithubAppID,
			"ARGOCD_GIT_GITHUB_APP_INSTALLATION_ID": em.ArgocdGitGithubAppInstallationID,
			"ARGOCD_GIT_GITHUB_APP_PRIVATE_KEY":     em.ArgocdGitGithubAppPrivateKey,
		}); err != nil {
			return err
		}
		return validateHTTPGitURL(em.ArgocdGitUrl, GitAuthModeGitHubApp)
	default:
		return &ErrorEnvMap{
			Message: fmt.Sprintf("Invalid ARGOCD_GIT_AUTH_MODE %q. Supported values: %s, %s, %s", em.ArgocdGitAuthMode, GitAuthModeHTTPS, GitAuthModeSSH, GitAuthModeGitHubApp),
			Err:     ErrInvalidEnvValue,
		}
	}
}

func validateRequiredEnvValues(values map[string]string) error {
	var missing []string
	for name, value := range values {
		if !IsConfiguredEnvValue(value) {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)

	return &ErrorEnvMap{
		Message: fmt.Sprintf("Vars not set: %+v", missing),
		Err:     ErrEnvsNotSet,
	}
}

func validateSSHGitURL(value string) error {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "ssh://") || (!strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && strings.Contains(trimmed, "@")) {
		return nil
	}

	return &ErrorEnvMap{
		Message: "ARGOCD_GIT_AUTH_MODE=ssh requires ARGOCD_GIT_URL to be an SSH repository URL such as git@github.com:org/repo.git or ssh://git@example.com/org/repo.git",
		Err:     ErrInvalidEnvValue,
	}
}

func validateHTTPGitURL(value, mode string) error {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return nil
	}

	return &ErrorEnvMap{
		Message: fmt.Sprintf("ARGOCD_GIT_AUTH_MODE=%s requires ARGOCD_GIT_URL to be an HTTP(S) repository URL", mode),
		Err:     ErrInvalidEnvValue,
	}
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

// GitAuthMode returns the configured Argo CD Git auth mode.
// Empty values keep the legacy HTTPS username + PAT/password behavior.
func (em *EnvMap) GitAuthMode() string {
	mode := strings.ToLower(strings.TrimSpace(em.ArgocdGitAuthMode))
	if mode == "" || mode == "<...>" {
		return GitAuthModeHTTPS
	}
	return mode
}

// GitRepositoryURL returns the preferred repository URL for Argo CD.
// ARGOCD_GIT_HTTPS_URL is a legacy fallback for existing .env files.
func (em *EnvMap) GitRepositoryURL() string {
	if IsConfiguredEnvValue(em.ArgocdGitUrl) {
		return strings.TrimSpace(em.ArgocdGitUrl)
	}
	return strings.TrimSpace(em.ArgocdGitHttpsUrl)
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
