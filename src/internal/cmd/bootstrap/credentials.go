package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/rs/zerolog/log"
)

// Credentials holds bootstrap secrets for Argo CD Git/Helm repositories and image registries.
type Credentials struct {
	GitUsername                   string
	GitPassword                   string
	GitSSHPrivateKey              string
	GitGitHubAppID                string
	GitGitHubAppInstallationID    string
	GitGitHubAppPrivateKey        string
	GitGitHubAppEnterpriseBaseURL string
	HelmRepoUsername              string
	HelmRepoPassword              string
	DockerconfigBase64            string
	ArgoCDPassword                string
}

var removedVariables = map[string]string{
	"PROJECT_NAME":         "project name is not used anymore. define each cluster name in config.yaml",
	"PROJECT_STAGE":        "project stage is not used anymore. define per cluster in config.yaml",
	"ARGOCD_GIT_URL":       "repository URL is configured in config.yaml",
	"ARGOCD_GIT_HTTPS_URL": "repository URL is configured in config.yaml",
	"ARGOCD_HELM_REPO_URL": "Helm repository URL is configured in config.yaml",
}

// LoadCredentials loads bootstrap credentials from the OS environment and an optional .env file in workDir.
// OS environment variables always take precedence over values defined in .env.
// Precedence for each credential:
//  1. OS canonical (GITOPS_*)
//  2. OS legacy alias (ARGOCD_*) [emits deprecation warning]
//  3. .env canonical (GITOPS_*)
//  4. .env legacy alias (ARGOCD_*) [emits deprecation warning]
func LoadCredentials(workDir string) (*Credentials, error) {
	envFilePath := filepath.Join(workDir, ".env")
	fileVars := make(map[string]string)

	if vars, err := godotenv.Read(envFilePath); err == nil {
		fileVars = vars
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read .env file at %q: %w", envFilePath, err)
	}

	warnRemovedVariables(fileVars)

	creds := &Credentials{
		GitUsername:                   resolveVar(fileVars, "GITOPS_USERNAME", "ARGOCD_GIT_USERNAME"),
		GitPassword:                   resolveVar(fileVars, "GITOPS_PASSWORD", "GITOPS_PAT_OR_PASSWORD", "ARGOCD_GIT_PASSWORD", "ARGOCD_GIT_PAT_OR_PASSWORD"),
		GitGitHubAppID:                resolveVar(fileVars, "GITOPS_GITHUB_APP_ID", "ARGOCD_GIT_GITHUB_APP_ID"),
		GitGitHubAppInstallationID:    resolveVar(fileVars, "GITOPS_GITHUB_APP_INSTALLATION_ID", "ARGOCD_GIT_GITHUB_APP_INSTALLATION_ID"),
		GitGitHubAppEnterpriseBaseURL: resolveVar(fileVars, "GITOPS_GITHUB_APP_ENTERPRISE_BASE_URL", "ARGOCD_GIT_GITHUB_APP_ENTERPRISE_BASE_URL"),
		HelmRepoUsername:              resolveVar(fileVars, "GITOPS_HELM_REPO_USERNAME", "ARGOCD_HELM_REPO_USERNAME"),
		HelmRepoPassword:              resolveVar(fileVars, "GITOPS_HELM_REPO_PASSWORD", "ARGOCD_HELM_REPO_PASSWORD"),
		DockerconfigBase64:            resolveVar(fileVars, "DOCKERCONFIG_BASE64"),
		ArgoCDPassword:                resolveVar(fileVars, "ARGOCD_PASSWORD", "ARGOCD_WIZARD_ACCOUNT_PASSWORD"),
	}

	sshKey, err := resolveKeyOrPath(workDir, fileVars, "GITOPS_SSH_PRIVATE_KEY", "ARGOCD_GIT_SSH_PRIVATE_KEY", "GITOPS_SSH_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}
	creds.GitSSHPrivateKey = sshKey

	githubAppKey, err := resolveKeyOrPath(workDir, fileVars, "GITOPS_GITHUB_APP_PRIVATE_KEY", "ARGOCD_GIT_GITHUB_APP_PRIVATE_KEY", "GITOPS_GITHUB_APP_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}
	creds.GitGitHubAppPrivateKey = githubAppKey

	return creds, nil
}

// ValidateForCluster validates the credentials against the cluster's repository configurations.
func (c *Credentials) ValidateForCluster(cluster *config.Cluster) error {
	if cluster == nil {
		return nil
	}
	gitURL := ""
	if cluster.ArgoCD.Repo.Git != nil {
		gitURL = cluster.ArgoCD.Repo.Git.Configs.URL
	}
	trimmedURL := strings.TrimSpace(gitURL)

	switch {
	case utils.IsSSHGitURL(trimmedURL):
		if c.GitSSHPrivateKey == "" {
			return errors.New("SSH git repository URL requires GITOPS_SSH_PRIVATE_KEY or GITOPS_SSH_PRIVATE_KEY_PATH")
		}
	case c.GitSSHPrivateKey != "":
		if !utils.IsSSHGitURL(trimmedURL) {
			return fmt.Errorf("GITOPS_SSH_PRIVATE_KEY is configured, but git repository URL %q is not an SSH URL", trimmedURL)
		}
	case c.GitGitHubAppID != "":
		if c.GitGitHubAppInstallationID == "" || c.GitGitHubAppPrivateKey == "" {
			return errors.New("GitHub App authentication requires GITOPS_GITHUB_APP_ID, GITOPS_GITHUB_APP_INSTALLATION_ID, and GITOPS_GITHUB_APP_PRIVATE_KEY")
		}
		if !utils.IsHTTPGitURL(trimmedURL) {
			return fmt.Errorf("GitHub App authentication requires an HTTP(S) git URL, got %q", trimmedURL)
		}
	default:
		hasUsername := c.GitUsername != ""
		hasPassword := c.GitPassword != ""
		if hasUsername != hasPassword {
			return errors.New("GITOPS_USERNAME and GITOPS_PASSWORD must either both be set for a private repository or both be omitted for a public repository")
		}
		if hasUsername && !utils.IsHTTPGitURL(trimmedURL) {
			return fmt.Errorf("username/password authentication requires an HTTP(S) git URL, got %q", trimmedURL)
		}
	}

	if cluster.ArgoCD.HelmRepo != nil && cluster.ArgoCD.HelmRepo.URL != "" {
		hasHelmUser := c.HelmRepoUsername != ""
		hasHelmPass := c.HelmRepoPassword != ""
		if hasHelmUser != hasHelmPass {
			return errors.New("GITOPS_HELM_REPO_USERNAME and GITOPS_HELM_REPO_PASSWORD must either both be set for a private Helm repository or both be omitted")
		}
	}

	if c.DockerconfigBase64 != "" {
		if _, err := utils.DecodeB64(c.DockerconfigBase64); err != nil {
			return fmt.Errorf("invalid DOCKERCONFIG_BASE64: %w", err)
		}
	}

	return nil
}

func warnRemovedVariables(fileVars map[string]string) {
	for varName, reason := range removedVariables {
		if v, ok := os.LookupEnv(varName); ok && isConfigured(v) {
			log.Warn().Str("var", varName).Msgf("ignoring removed environment variable %s: %s", varName, reason)
		} else if v, ok := fileVars[varName]; ok && isConfigured(v) {
			log.Warn().Str("var", varName).Msgf("ignoring removed variable %s in .env: %s", varName, reason)
		}
	}
}

func resolveVar(fileVars map[string]string, canonical string, legacyAliases ...string) string {
	if v, ok := os.LookupEnv(canonical); ok && isConfigured(v) {
		return strings.TrimSpace(v)
	}
	for _, alias := range legacyAliases {
		if v, ok := os.LookupEnv(alias); ok && isConfigured(v) {
			log.Warn().Str("deprecated", alias).Str("replacement", canonical).
				Msgf("environment variable %s is deprecated; please use %s instead", alias, canonical)
			return strings.TrimSpace(v)
		}
	}
	if v, ok := fileVars[canonical]; ok && isConfigured(v) {
		return strings.TrimSpace(v)
	}
	for _, alias := range legacyAliases {
		if v, ok := fileVars[alias]; ok && isConfigured(v) {
			log.Warn().Str("deprecated", alias).Str("replacement", canonical).
				Msgf("variable %s in .env is deprecated; please use %s instead", alias, canonical)
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func resolveKeyOrPath(workDir string, fileVars map[string]string, inlineCanonical string, inlineLegacy string, pathKey string) (string, error) {
	inlineVal := resolveVar(fileVars, inlineCanonical, inlineLegacy)
	pathVal := resolveVar(fileVars, pathKey)

	if inlineVal != "" && pathVal != "" {
		return "", fmt.Errorf("cannot configure both %s and %s", inlineCanonical, pathKey)
	}

	if pathVal != "" {
		keyFilePath, err := utils.GetFullPath(pathVal, workDir)
		if err != nil {
			return "", fmt.Errorf("resolve key file path %q: %w", pathVal, err)
		}
		content, err := os.ReadFile(keyFilePath)
		if err != nil {
			return "", fmt.Errorf("read key file %q specified by %s: %w", keyFilePath, pathKey, err)
		}
		return strings.TrimSpace(string(content)), nil
	}

	return inlineVal, nil
}

func isConfigured(v string) bool {
	trimmed := strings.TrimSpace(v)
	return trimmed != "" && trimmed != "<...>"
}
