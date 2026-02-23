package envmap

import (
	"errors"
	"fmt"
	"kubara/utils"
	"reflect"
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
	_                           struct{} `doc:"# 💡 Dummy values (without <>) are optional and can be left as-is if not needed"`
	_                           struct{} `doc:"#    (e.g. no private image registry). It will still create a secret, but it will be not valid."`
	_                           struct{} `doc:"\n### Project related values"`
	ProjectName                 string   `default:"<...>" koanf:"PROJECT_NAME"`
	ProjectStage                string   `default:"<...>" koanf:"PROJECT_STAGE"`
	_                           struct{} `doc:"\n### STACKIT Portal related values"`
	_                           struct{} `doc:"\n### Docker related values"`
	_                           struct{} `doc:"# see https://docs.docker.com/reference/cli/docker/login/"`
	_                           struct{} `doc:"# after successful login you can look inside envMap.json in your docker directory (~/.docker/envMap.json) on Linux/Mac"`
	_                           struct{} `doc:"# the variable must be base64 encoded - how to: https://docs.kubara.io/latest-stable/6_reference/faq/#how-do-i-create-a-dockerconfigjson-for-env-file"`
	DockerconfigBase64          string   `default:"<...>" koanf:"DOCKERCONFIG_BASE64"`
	_                           struct{} `doc:"\n### Argo CD related values"`
	ArgocdWizardAccountPassword string   `default:"<...>" koanf:"ARGOCD_WIZARD_ACCOUNT_PASSWORD"`
	_                           struct{} `doc:"\n### Helm repository values"`
	ArgocdHelmRepoUsername      string   `default:"<...>" koanf:"ARGOCD_HELM_REPO_USERNAME"`
	ArgocdHelmRepoPassword      string   `default:"<...>" koanf:"ARGOCD_HELM_REPO_PASSWORD"`
	ArgocdHelmRepoUrl           string   `default:"<...>" koanf:"ARGOCD_HELM_REPO_URL"`
	_                           struct{} `doc:"\n### Git repository values"`
	ArgocdGitHttpsUrl           string   `default:"<...>" koanf:"ARGOCD_GIT_HTTPS_URL"`
	ArgocdGitPatOrPassword      string   `default:"<...>" koanf:"ARGOCD_GIT_PAT_OR_PASSWORD"`
	ArgocdGitUsername           string   `default:"<...>" koanf:"ARGOCD_GIT_USERNAME"`
	_                           struct{} `doc:"\n### DNS Name/Zones related values"`
	_                           struct{} `doc:"# The Domain name under which your dns-entries will be added."`
	_                           struct{} `doc:"# The resulting dnsZone name will be a concatenation of <PROJECT_NAME>-<PROJECT_STAGE>.<DOMAIN_NAME>"`
	_                           struct{} `doc:"# the value should be looking like 'stackit.zone' eg. 'yourDomain.com'"`
	DomainName                  string   `default:"<...>" koanf:"DOMAIN_NAME"`
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
		if utils.IsDefaultValue(field, defaultTagVal) {
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
