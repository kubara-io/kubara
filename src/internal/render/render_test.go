package render

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTemplatesFS = catalog.BuiltInFS()

// helper function to setup test filesystem with correct root path
func setupTestFS(_ *testing.T) func() {
	originalFS := templatesFSNew
	templatesFSNew = testTemplatesFS

	// Return cleanup function
	return func() {
		templatesFSNew = originalFS
	}
}

func fullServiceContext() map[string]any {
	return map[string]any{
		"argocd":                  map[string]any{"status": "enabled"},
		"cert-manager":            map[string]any{"status": "enabled", "config": map[string]any{"clusterIssuer": map[string]any{"name": "letsencrypt-prod", "email": "admin@example.com", "server": "https://acme-staging-v02.api.letsencrypt.org/directory"}}},
		"external-dns":            map[string]any{"status": "enabled"},
		"external-secrets":        map[string]any{"status": "enabled"},
		"kube-prometheus-stack":   map[string]any{"status": "enabled"},
		"traefik":                 map[string]any{"status": "enabled"},
		"kyverno":                 map[string]any{"status": "enabled"},
		"kyverno-policies":        map[string]any{"status": "enabled"},
		"kyverno-policy-reporter": map[string]any{"status": "enabled"},
		"loki":                    map[string]any{"status": "enabled"},
		"homer-dashboard":         map[string]any{"status": "enabled"},
		"oauth2-proxy":            map[string]any{"status": "disabled"},
		"metrics-server":          map[string]any{"status": "disabled"},
		"metallb":                 map[string]any{"status": "disabled"},
		"longhorn":                map[string]any{"status": "disabled"},
	}
}

func fullCatalogContext() map[string]any {
	return map[string]any{
		"services": map[string]any{
			"argocd":                  map[string]any{"chartPath": "argo-cd"},
			"cert-manager":            map[string]any{"chartPath": "cert-manager"},
			"external-dns":            map[string]any{"chartPath": "external-dns"},
			"external-secrets":        map[string]any{"chartPath": "external-secrets"},
			"kube-prometheus-stack":   map[string]any{"chartPath": "kube-prometheus-stack"},
			"traefik":                 map[string]any{"chartPath": "traefik"},
			"kyverno":                 map[string]any{"chartPath": "kyverno"},
			"kyverno-policies":        map[string]any{"chartPath": "kyverno-policies"},
			"kyverno-policy-reporter": map[string]any{"chartPath": "kyverno-policy-reporter"},
			"loki":                    map[string]any{"chartPath": "loki"},
			"homer-dashboard":         map[string]any{"chartPath": "homer-dashboard"},
			"oauth2-proxy":            map[string]any{"chartPath": "oauth2-proxy"},
			"metrics-server":          map[string]any{"chartPath": "metrics-server"},
			"metallb":                 map[string]any{"chartPath": "metallb"},
			"longhorn":                map[string]any{"chartPath": "longhorn"},
		},
	}
}

func TestTemplateType_String(t *testing.T) {
	tests := []struct {
		name string
		tt   TemplateType
		want string
	}{
		{
			name: "Terraform type returns correct string",
			tt:   Terraform,
			want: "terraform",
		},
		{
			name: "Helm type returns correct string",
			tt:   Helm,
			want: "helm",
		},
		{
			name: "All type returns correct string",
			tt:   All,
			want: "all",
		},
		{
			name: "Invalid type returns empty string",
			tt:   TemplateType(99),
			want: "", // Falls back to empty since not in map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tt.String())
		})
	}
}

func TestMakeWalkDirFunc(t *testing.T) {
	// Create test filesystem structure
	testFS := testTemplatesFS

	var files []string
	walkFunc := makeWalkDirFunc(tmplRoot, &files)

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	require.NoError(t, err)

	// Verify that files are collected (not directories)
	require.NotEmpty(t, files)
	for _, file := range files {
		assert.NotEmpty(t, file)
		assert.False(t, strings.HasSuffix(file, "/"))
	}

	// Test error propagation if WalkDir encounters an error
	var errorFiles []string
	errorWalkFunc := makeWalkDirFunc(tmplRoot, &errorFiles)
	// Intentionally walk non-existent path to trigger error
	err = fs.WalkDir(testFS, "nonexistent", errorWalkFunc)
	assert.Error(t, err)
	assert.Empty(t, errorFiles)
}

func TestMakeWalkDirFunc_RelPathError(t *testing.T) {
	// Test relative path error (edge case: path outside root)
	testFS := testTemplatesFS
	var files []string
	walkFunc := makeWalkDirFunc("nonexistent-root", &files) // Invalid root

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	// Should still work but paths might be relative to nonexistent root
	require.NoError(t, err)
}

func TestMakeWalkDirFunc_DirectoryFiltering(t *testing.T) {
	// Test that directories are properly filtered out
	testFS := testTemplatesFS
	var files []string
	walkFunc := makeWalkDirFunc(tmplRoot, &files)

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	require.NoError(t, err)

	// Ensure no directory entries (ending with /) are included
	for _, file := range files {
		assert.False(t, strings.HasSuffix(file, "/"), "File path should not end with /: %s", file)
	}
}

func TestSelectTemplatesForProvider_PrefersProviderSpecificFile(t *testing.T) {
	files := []string{
		"managed-service-catalog/terraform/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/t-cloud-public/modules/iam/main.tf",
		"managed-service-catalog/terraform/modules/iam/variables.tf",
	}

	selected := selectTemplatesForProvider(files, "stackit")

	assert.Contains(t, selected, "managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/providers/t-cloud-public/modules/iam/main.tf")
	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/variables.tf")
	require.Len(t, selected, 2)
}

func TestSelectTemplatesForProvider_FallsBackToCommonFile(t *testing.T) {
	files := []string{
		"managed-service-catalog/terraform/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf",
		"managed-service-catalog/terraform/modules/iam/variables.tf",
	}

	selected := selectTemplatesForProvider(files, "azure")

	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf")
	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/variables.tf")
	require.Len(t, selected, 2)
}

func TestStripProviderPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips providers/stackit under terraform",
			input: "customer-service-catalog/terraform/providers/stackit/example/infrastructure/main.tf.tplt",
			want:  "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt",
		},
		{
			name:  "strips providers/stackit under managed terraform",
			input: "managed-service-catalog/terraform/providers/stackit/modules/ske-cluster/main.tf",
			want:  "managed-service-catalog/terraform/modules/ske-cluster/main.tf",
		},
		{
			name:  "leaves non-provider terraform path unchanged",
			input: "managed-service-catalog/terraform/images/public-cloud-0.png",
			want:  "managed-service-catalog/terraform/images/public-cloud-0.png",
		},
		{
			name:  "does not strip providers/<name> outside terraform or helm context",
			input: "some-catalog/providers/stackit/file.txt",
			want:  "some-catalog/providers/stackit/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripProviderPath(tt.input))
		})
	}
}

func TestTemplateFiles_TCloudPublicProviderSelectsCCEArtifacts(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var paths []string
	for _, result := range results {
		require.NoError(t, result.Error)
		paths = append(paths, result.Path)
	}

	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/dns-zone/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/keypair/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/kms-key/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/objectstorage-bucket/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/openbao-helm/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/storage-classes/main.tf")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/main.tf.tplt")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/main.tf.tplt")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/terraform.tf.tplt")
	assert.NotContains(t, paths, "managed-service-catalog/terraform/providers/stackit/modules/ske-cluster/main.tf")
}

func TestTemplateFiles_TCloudPublicAgenciesUseDefaultProvider(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var bootstrapMain string
	var infrastructureMain string
	var infrastructureProviders string
	var infrastructureVariables string
	var infrastructureEnv string
	var infrastructureOutputs string
	var agencyMain string
	var agencyVariables string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/main.tf.tplt":
			bootstrapMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt":
			infrastructureMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/terraform.tf.tplt":
			infrastructureProviders = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/variables.tf.tplt":
			infrastructureVariables = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			infrastructureEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/outputs.tf.tplt":
			infrastructureOutputs = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/main.tf":
			agencyMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/variables.tf":
			agencyVariables = result.Content
		}
	}

	require.NotEmpty(t, bootstrapMain)
	require.NotEmpty(t, infrastructureMain)
	require.NotEmpty(t, infrastructureProviders)
	require.NotEmpty(t, infrastructureVariables)
	require.NotEmpty(t, infrastructureEnv)
	require.NotEmpty(t, infrastructureOutputs)
	require.NotEmpty(t, agencyMain)
	require.NotEmpty(t, agencyVariables)

	for _, content := range []string{bootstrapMain, infrastructureMain, infrastructureProviders} {
		assert.NotContains(t, content, `alias       = "agency"`)
		assert.NotContains(t, content, "opentelekomcloud.agency")
	}
	assert.Contains(t, infrastructureProviders, "tenant_name = var.t_cloud_public_tenant_name")
	assert.Contains(t, infrastructureProviders, `alias       = "global-region"`)
	assert.Contains(t, infrastructureProviders, "tenant_name = var.t_cloud_public_region")
	assert.Contains(t, infrastructureProviders, "skip_metadata_api_check")
	assert.Contains(t, infrastructureProviders, `source  = "hashicorp/helm"`)
	assert.Contains(t, infrastructureProviders, `source  = "hashicorp/kubernetes"`)
	assert.Contains(t, infrastructureProviders, `provider "helm"`)
	assert.Contains(t, infrastructureProviders, `provider "kubernetes"`)
	assert.Contains(t, infrastructureProviders, "yamldecode(module.cce_cluster.kubeconfig_raw)")
	assert.Contains(t, bootstrapMain, `alias       = "global-region"`)
	assert.Contains(t, bootstrapMain, "tenant_name = var.t_cloud_public_region")
	assert.Contains(t, bootstrapMain, `module "bucket_kms_key"`)
	assert.Contains(t, bootstrapMain, "opentelekomcloud = opentelekomcloud.global-region")
	assert.Contains(t, bootstrapMain, "opentelekomcloud.global-region = opentelekomcloud.global-region")
	assert.Contains(t, infrastructureMain, "count  = length(local.t_cloud_public_agencies) > 0 ? 1 : 0")
	assert.Contains(t, infrastructureMain, "project = var.t_cloud_public_tenant_name")
	assert.Contains(t, infrastructureMain, "addons     = var.cce_addons")
	assert.Contains(t, infrastructureMain, `module "storage_classes"`)
	assert.Contains(t, infrastructureMain, "source = \"../../../../managed-service-catalog/terraform/modules/storage-classes\"")
	assert.Contains(t, infrastructureMain, `"everest.io/crypt-key-id" = module.node_storage_kms_key.id`)
	assert.Contains(t, infrastructureMain, `module "openbao"`)
	assert.Contains(t, infrastructureMain, "source = \"../../../../managed-service-catalog/terraform/modules/openbao-helm\"")
	assert.Contains(t, infrastructureMain, "depends_on = [module.cce_cluster]")
	assert.Contains(t, infrastructureMain, "ingress_enabled")
	assert.Contains(t, infrastructureMain, "var.openbao_ingress_enabled")
	assert.Contains(t, infrastructureMain, "var.openbao_ingress_host")
	assert.Contains(t, infrastructureMain, "load_balancer_type               = var.load_balancer_type")
	assert.Contains(t, infrastructureMain, "dedicated_load_balancer_availability_zones = var.dedicated_load_balancer_availability_zones")
	assert.NotContains(t, infrastructureMain, "cce_agency_projects")
	assert.NotContains(t, infrastructureVariables, "cce_agency_projects")
	assert.NotContains(t, infrastructureEnv, "cce_agency_projects")
	assert.Contains(t, infrastructureMain, "var.create_obs_kms_agency ? {")
	assert.Contains(t, infrastructureMain, "obs_kms = {")
	assert.Contains(t, infrastructureMain, "project = var.t_cloud_public_region")
	assert.Contains(t, infrastructureVariables, `variable "create_obs_kms_agency"`)
	assert.Contains(t, infrastructureVariables, `variable "cce_addons"`)
	assert.Contains(t, infrastructureVariables, `variable "create_storage_classes"`)
	assert.Contains(t, infrastructureVariables, `variable "storage_classes"`)
	assert.Contains(t, infrastructureVariables, `variable "enable_openbao"`)
	assert.Contains(t, infrastructureVariables, `variable "openbao_seal_config"`)
	assert.Contains(t, infrastructureVariables, `variable "openbao_ingress_enabled"`)
	assert.Contains(t, infrastructureVariables, `default     = "test.example.com"`)
	assert.Contains(t, infrastructureVariables, `default     = "/openbao"`)
	assert.Contains(t, infrastructureVariables, `variable "load_balancer_type"`)
	assert.Contains(t, infrastructureVariables, `variable "dedicated_load_balancer_availability_zones"`)
	assert.NotContains(t, infrastructureVariables, `variable "obs_kms_agency_propagation_delay"`)
	assert.Contains(t, infrastructureEnv, "create_obs_kms_agency")
	assert.Contains(t, infrastructureEnv, "= false")
	assert.Contains(t, infrastructureEnv, `load_balancer_type                           = "shared"`)
	assert.Contains(t, infrastructureEnv, `dedicated_load_balancer_availability_zones   = ["eu-de-01"]`)
	assert.Contains(t, infrastructureEnv, `dedicated_load_balancer_l4_flavor_name       = "L4_flavor.elb.s1.small"`)
	assert.Contains(t, infrastructureEnv, `dedicated_load_balancer_l7_flavor_name       = "L7_flavor.elb.s1.small"`)
	assert.Contains(t, infrastructureEnv, "cce_addons = {")
	assert.Contains(t, infrastructureEnv, "metrics-server = {")
	assert.Contains(t, infrastructureEnv, `version = "1.3.104"`)
	assert.NotContains(t, infrastructureEnv, "coredns = {")
	assert.NotContains(t, infrastructureEnv, "everest = {")
	assert.Contains(t, infrastructureEnv, "create_storage_classes = true")
	assert.Contains(t, infrastructureEnv, "csi-disk-retain-topology-crypt")
	assert.Contains(t, infrastructureEnv, "csi-obs-pfs-retain")
	assert.Contains(t, infrastructureEnv, "csi-disk-default")
	assert.Contains(t, infrastructureEnv, `"everest.io/disk-volume-type"        = "SSD"`)
	assert.NotContains(t, infrastructureEnv, `"everest.io/disk-volume-type"        = "SATA"`)
	assert.Contains(t, infrastructureEnv, `use_node_storage_kms_key = true`)
	assert.Contains(t, infrastructureEnv, "enable_openbao")
	assert.Contains(t, infrastructureEnv, `openbao_chart_version           = "0.28.3"`)
	assert.Contains(t, infrastructureEnv, `openbao_seal_config             = ""`)
	assert.Contains(t, infrastructureEnv, `openbao_ingress_enabled         = true`)
	assert.Contains(t, infrastructureEnv, `openbao_ingress_host            = "test.example.com"`)
	assert.Contains(t, infrastructureEnv, `openbao_ingress_path            = "/openbao"`)
	assert.Contains(t, infrastructureOutputs, `output "load_balancer_id"`)
	assert.Contains(t, infrastructureOutputs, "module.network.load_balancer_id")
	assert.Contains(t, infrastructureOutputs, `output "storage_classes"`)
	assert.Contains(t, infrastructureOutputs, `output "openbao_release_name"`)
	assert.Contains(t, infrastructureOutputs, `output "openbao_ingress_url"`)
	assert.Contains(t, agencyMain, "domain_roles          = try(length(each.value.domain_roles), 0) > 0 ? each.value.domain_roles : null")
	assert.Contains(t, agencyVariables, "domain_roles = optional(list(string))")
	assert.NotContains(t, agencyVariables, "domain_roles = optional(list(string), [])")
}

func TestTemplateFiles_TCloudPublicProviderRendersVeleroBucketWhenEnabled(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{
					"velero": map[string]any{"status": "enabled"},
				},
			},
		},
	})

	require.NoError(t, err)

	var mainContent string
	var envContent string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt":
			mainContent = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			envContent = result.Content
		}
	}

	require.NotEmpty(t, mainContent)
	require.NotEmpty(t, envContent)
	assert.Contains(t, mainContent, `module "velero_bucket"`)
	assert.Contains(t, mainContent, `module "velero_bucket_kms_key"`)
	assert.Contains(t, mainContent, "opentelekomcloud = opentelekomcloud.global-region")
	assert.Contains(t, mainContent, "opentelekomcloud.global-region = opentelekomcloud.global-region")
	assert.Contains(t, mainContent, "depends_on = [module.t_cloud_public_agencies]")
	assert.Contains(t, envContent, "velero_bucket_name")
}

func TestTemplateFiles_TCloudPublicEnvAutoTfvarsDoesNotRenderProviderCredentials(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var infrastructureEnv string
	var bootstrapEnv string
	var infrastructureVariables string
	var setEnvSh string
	var setEnvPS1 string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			infrastructureEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/env.auto.tfvars.tplt":
			bootstrapEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/variables.tf.tplt":
			infrastructureVariables = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/set-env-changeme.sh.tplt":
			setEnvSh = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/set-env-changeme.ps1.tplt":
			setEnvPS1 = result.Content
		}
	}

	require.NotEmpty(t, infrastructureEnv)
	require.NotEmpty(t, bootstrapEnv)
	require.NotEmpty(t, infrastructureVariables)
	require.NotEmpty(t, setEnvSh)
	require.NotEmpty(t, setEnvPS1)

	for _, content := range []string{infrastructureEnv, bootstrapEnv} {
		assert.NotContains(t, content, "t_cloud_public_region")
		assert.NotContains(t, content, "t_cloud_public_domain_name")
		assert.NotContains(t, content, "t_cloud_public_tenant_name")
		assert.NotContains(t, content, "t_cloud_public_access_key")
		assert.NotContains(t, content, "t_cloud_public_secret_key")
	}
	assert.Contains(t, infrastructureVariables, `default     = "test-tenant"`)
	assert.Contains(t, setEnvSh, `export AWS_REQUEST_CHECKSUM_CALCULATION="when_required"`)
	assert.Contains(t, setEnvSh, `export AWS_RESPONSE_CHECKSUM_VALIDATION="when_required"`)
	assert.Contains(t, setEnvPS1, `$env:AWS_REQUEST_CHECKSUM_CALCULATION = "when_required"`)
	assert.Contains(t, setEnvPS1, `$env:AWS_RESPONSE_CHECKSUM_VALIDATION = "when_required"`)
}

func TestTemplateFiles_TCloudPublicBootstrapCredentialsAreSensitive(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var bootstrapMain string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/main.tf.tplt" {
			bootstrapMain = result.Content
		}
	}

	require.NotEmpty(t, bootstrapMain)
	accessKeyOutputIndex := strings.Index(bootstrapMain, `output "credential_access_key" {`)
	require.NotEqual(t, -1, accessKeyOutputIndex)

	accessKeyOutput := bootstrapMain[accessKeyOutputIndex:]
	accessKeyOutputEnd := strings.Index(accessKeyOutput, "\n}")
	require.NotEqual(t, -1, accessKeyOutputEnd)

	assert.Contains(t, accessKeyOutput[:accessKeyOutputEnd], "sensitive   = true")
}

func TestTemplateFiles_TCloudPublicOBSBucketCanBeDestroyed(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var objectstorageMain string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "managed-service-catalog/terraform/providers/t-cloud-public/modules/objectstorage-bucket/main.tf" {
			objectstorageMain = result.Content
		}
	}

	require.NotEmpty(t, objectstorageMain)
	bucketResourceIndex := strings.Index(objectstorageMain, `resource "opentelekomcloud_obs_bucket" "this" {`)
	require.NotEqual(t, -1, bucketResourceIndex)

	bucketResource := objectstorageMain[bucketResourceIndex:]
	bucketPolicyIndex := strings.Index(bucketResource, `resource "opentelekomcloud_obs_bucket_policy"`)
	require.NotEqual(t, -1, bucketPolicyIndex)

	assert.NotContains(t, bucketResource[:bucketPolicyIndex], "prevent_destroy = true")
}

func TestTemplateFiles_TCloudPublicManagedModulesDoNotPreventDestroy(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	for _, result := range results {
		require.NoError(t, result.Error)
		if strings.HasPrefix(result.Path, "managed-service-catalog/terraform/providers/t-cloud-public/modules/") {
			assert.NotContains(t, result.Content, "prevent_destroy", result.Path)
		}
	}
}

func TestTemplateFiles_TCloudPublicStorageClassesUseKMSKey(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var moduleMain string
	var moduleVariables string
	var moduleTerraform string
	var infrastructureMain string
	var infrastructureEnv string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/storage-classes/main.tf":
			moduleMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/storage-classes/variables.tf":
			moduleVariables = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/storage-classes/terraform.tf":
			moduleTerraform = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt":
			infrastructureMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			infrastructureEnv = result.Content
		}
	}

	require.NotEmpty(t, moduleMain)
	require.NotEmpty(t, moduleVariables)
	require.NotEmpty(t, moduleTerraform)
	require.NotEmpty(t, infrastructureMain)
	require.NotEmpty(t, infrastructureEnv)

	assert.Contains(t, moduleTerraform, `source  = "hashicorp/kubernetes"`)
	assert.Contains(t, moduleMain, `resource "kubernetes_storage_class_v1" "this"`)
	assert.Contains(t, moduleMain, `storage_provisioner    = each.value.storage_provisioner`)
	assert.Contains(t, moduleMain, `allow_volume_expansion = each.value.allow_volume_expansion`)
	assert.Contains(t, moduleVariables, `parameters             = map(string)`)
	assert.Contains(t, infrastructureMain, `"everest.io/crypt-key-id" = module.node_storage_kms_key.id`)
	assert.NotContains(t, infrastructureEnv, `everest.io/crypt-key-id`)
	assert.Contains(t, infrastructureEnv, `use_node_storage_kms_key = true`)
	assert.Contains(t, infrastructureEnv, `volume_binding_mode      = "Immediate"`)
}

func TestTemplateFiles_TCloudPublicCCEClusterRendersAddons(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var cceMain string
	var cceVariables string
	var cceOutputs string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/main.tf":
			cceMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/variables.tf":
			cceVariables = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/outputs.tf":
			cceOutputs = result.Content
		}
	}

	require.NotEmpty(t, cceMain)
	require.NotEmpty(t, cceVariables)
	require.NotEmpty(t, cceOutputs)

	assert.Contains(t, cceMain, `resource "opentelekomcloud_cce_addon_v3" "this"`)
	assert.Contains(t, cceMain, "for_each = local.enabled_addons")
	assert.Contains(t, cceMain, "template_name    = each.key")
	assert.Contains(t, cceMain, "template_version = each.value.version")
	assert.Contains(t, cceMain, "swr_addr = local.addon_image_endpoint")
	assert.Contains(t, cceVariables, `variable "addons"`)
	assert.Contains(t, cceOutputs, `output "addons"`)
}

func TestTemplateFiles_TCloudPublicNetworkSupportsDedicatedLoadBalancer(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var networkMain string
	var networkVariables string
	var networkOutputs string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/main.tf":
			networkMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/variables.tf":
			networkVariables = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/outputs.tf":
			networkOutputs = result.Content
		}
	}

	require.NotEmpty(t, networkMain)
	require.NotEmpty(t, networkVariables)
	require.NotEmpty(t, networkOutputs)
	assert.Contains(t, networkMain, `create_shared_load_balancer    = var.enable_load_balancer && var.load_balancer_type == "shared"`)
	assert.Contains(t, networkMain, `create_dedicated_load_balancer = var.enable_load_balancer && var.load_balancer_type == "dedicated"`)
	assert.Contains(t, networkMain, `resource "opentelekomcloud_lb_loadbalancer_v2" "this"`)
	assert.Contains(t, networkMain, `resource "opentelekomcloud_lb_loadbalancer_v3" "dedicated"`)
	assert.Contains(t, networkMain, `network_ids        = [opentelekomcloud_vpc_subnet_v1.this.network_id]`)
	assert.Contains(t, networkMain, `vip_subnet_id = opentelekomcloud_vpc_subnet_v1.this.subnet_id`)
	assert.Contains(t, networkMain, `availability_zones = var.dedicated_load_balancer_availability_zones`)
	assert.Contains(t, networkMain, `l4_flavor          = var.dedicated_load_balancer_l4_flavor_name != "" ? data.opentelekomcloud_lb_flavor_v3.dedicated_l4[0].id : null`)
	assert.Contains(t, networkMain, `l7_flavor          = var.dedicated_load_balancer_l7_flavor_name != "" ? data.opentelekomcloud_lb_flavor_v3.dedicated_l7[0].id : null`)
	assert.Contains(t, networkVariables, `default     = "shared"`)
	assert.Contains(t, networkVariables, `default     = ["eu-de-01"]`)
	assert.Contains(t, networkVariables, `default     = "L4_flavor.elb.s1.small"`)
	assert.Contains(t, networkVariables, `default     = "L7_flavor.elb.s1.small"`)
	assert.Contains(t, networkOutputs, `opentelekomcloud_lb_loadbalancer_v3.dedicated[0].id`)
	assert.Contains(t, networkOutputs, `opentelekomcloud_vpc_eip_v1.dedicated_load_balancer[0].publicip[0].ip_address`)
	assert.Contains(t, networkOutputs, `value       = opentelekomcloud_vpc_subnet_v1.this.network_id`)
}

func TestTemplateFiles_TCloudPublicOpenBaoModuleUsesHelmRelease(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var openbaoMain string
	var openbaoVariables string
	var openbaoTerraform string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/openbao-helm/main.tf":
			openbaoMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/openbao-helm/variables.tf":
			openbaoVariables = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/openbao-helm/terraform.tf":
			openbaoTerraform = result.Content
		}
	}

	require.NotEmpty(t, openbaoMain)
	require.NotEmpty(t, openbaoVariables)
	require.NotEmpty(t, openbaoTerraform)
	assert.Contains(t, openbaoTerraform, `source  = "hashicorp/helm"`)
	assert.Contains(t, openbaoMain, `resource "helm_release" "this"`)
	assert.Contains(t, openbaoMain, `repository       = var.repository`)
	assert.Contains(t, openbaoMain, `chart            = var.chart`)
	assert.Contains(t, openbaoMain, `create_namespace = true`)
	assert.Contains(t, openbaoMain, `wait             = false`)
	assert.NotContains(t, openbaoMain, `timeout`)
	assert.Contains(t, openbaoMain, `ha = {`)
	assert.Contains(t, openbaoMain, `raft = {`)
	assert.Contains(t, openbaoMain, `setNodeId = true`)
	assert.Contains(t, openbaoMain, `${trimspace(var.seal_config)}`)
	assert.Contains(t, openbaoMain, `apiAddr  = local.ingress_url`)
	assert.Contains(t, openbaoMain, `ingress = {`)
	assert.Contains(t, openbaoMain, `"traefik.ingress.kubernetes.io/app-root"`)
	assert.Contains(t, openbaoMain, `"traefik.ingress.kubernetes.io/router.middlewares"`)
	assert.Contains(t, openbaoMain, `extraObjects = local.ingress_extra_objects`)
	assert.Contains(t, openbaoMain, `replacePathRegex`)
	assert.Contains(t, openbaoVariables, `default     = "https://openbao.github.io/openbao-helm"`)
	assert.Contains(t, openbaoVariables, `default     = "0.28.3"`)
	assert.Contains(t, openbaoVariables, `variable "ingress_enabled"`)
	assert.Contains(t, openbaoVariables, `default     = "/openbao"`)
}

func TestTemplateFiles_TCloudPublicOpenBaoLayerConfiguresSecrets(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
			"env": map[string]any{
				"DockerconfigBase64": "e30=",
			},
		},
	})

	require.NoError(t, err)

	var openbaoTerraform string
	var openbaoMain string
	var openbaoVariables string
	var openbaoEnv string
	var openbaoOutputs string
	var openbaoSecretsExample string
	var setEnvSh string
	var setEnvPs1 string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/set-env-changeme.sh.tplt":
			setEnvSh = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/set-env-changeme.ps1.tplt":
			setEnvPs1 = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/terraform.tf.tplt":
			openbaoTerraform = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/main.tf.tplt":
			openbaoMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/variables.tf.tplt":
			openbaoVariables = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/env.auto.tfvars.tplt":
			openbaoEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/outputs.tf.tplt":
			openbaoOutputs = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/secrets.auto.tfvars.example.tplt":
			openbaoSecretsExample = result.Content
		}
	}

	require.NotEmpty(t, openbaoTerraform)
	require.NotEmpty(t, openbaoMain)
	require.NotEmpty(t, openbaoVariables)
	require.NotEmpty(t, openbaoEnv)
	require.NotEmpty(t, openbaoOutputs)
	require.NotEmpty(t, openbaoSecretsExample)
	require.NotEmpty(t, setEnvSh)
	require.NotEmpty(t, setEnvPs1)
	assert.Contains(t, openbaoTerraform, `key    = "tf-state-test-cluster-dev-openbao"`)
	assert.Contains(t, openbaoTerraform, `source  = "hashicorp/vault"`)
	assert.Contains(t, openbaoTerraform, `provider "vault"`)
	assert.Contains(t, openbaoMain, `resource "vault_mount" "kv"`)
	assert.Contains(t, openbaoMain, `resource "vault_jwt_auth_backend" "oidc"`)
	assert.Contains(t, openbaoMain, `resource "vault_auth_backend" "kubernetes"`)
	assert.Contains(t, openbaoMain, `resource "vault_kubernetes_auth_backend_config" "kubernetes"`)
	assert.Contains(t, openbaoMain, `resource "vault_kubernetes_auth_backend_role" "external_secrets"`)
	assert.Contains(t, openbaoMain, `resource "vault_policy" "external_secrets_read"`)
	assert.Contains(t, openbaoMain, `resource "vault_policy" "kubernetes_namespace_kv_read"`)
	assert.Contains(t, openbaoMain, `path "sys/*"`)
	assert.Contains(t, openbaoMain, `oidc_scopes           = var.openbao_oidc_scopes`)
	assert.Contains(t, openbaoMain, `token_ttl             = var.openbao_oidc_admin_token_ttl`)
	assert.Contains(t, openbaoMain, `token_ttl                        = var.openbao_namespace_kv_read_token_ttl`)
	assert.Contains(t, openbaoMain, `identity.entity.aliases.${vault_auth_backend.kubernetes[0].accessor}.metadata.service_account_namespace`)
	assert.Contains(t, openbaoMain, `resource "vault_generic_endpoint" "external_secrets_user"`)
	assert.Contains(t, openbaoMain, `resource "vault_kv_secret_v2" "t_cloud_public_clouds_yaml"`)
	assert.Contains(t, openbaoMain, `name  = "t-cloud-public-clouds-yaml"`)
	assert.NotContains(t, openbaoMain, `argocd_customer_repository_credentials`)
	assert.NotContains(t, openbaoMain, `argocd_managed_repository_credentials`)
	assert.Contains(t, openbaoVariables, `variable "openbao_token"`)
	assert.Contains(t, openbaoVariables, `variable "openbao_oidc_discovery_url"`)
	assert.Contains(t, openbaoVariables, `https://test.example.com/openbao/ui/vault/auth/oidc/oidc/callback`)
	assert.Contains(t, openbaoVariables, `http://127.0.0.1:8200/ui/vault/auth/oidc/oidc/callback`)
	assert.NotContains(t, openbaoVariables, `CHANGE_ME_OPENBAO_DNS_NAME`)
	assert.Contains(t, openbaoVariables, `variable "openbao_oidc_scopes"`)
	assert.Contains(t, openbaoVariables, `default     = ["openid", "email", "profile"]`)
	assert.Contains(t, openbaoVariables, `variable "openbao_oidc_admin_token_ttl"`)
	assert.Contains(t, openbaoVariables, `default     = 604800`)
	assert.Contains(t, openbaoVariables, `variable "openbao_kubernetes_auth_path"`)
	assert.Contains(t, openbaoVariables, `variable "openbao_namespace_kv_read_token_ttl"`)
	assert.Contains(t, openbaoVariables, `default     = 86400`)
	assert.NotContains(t, openbaoVariables, `argocd_customer_repository`)
	assert.NotContains(t, openbaoVariables, `argocd_managed_repository`)
	assert.Contains(t, openbaoVariables, `sensitive   = true`)
	assert.Contains(t, openbaoEnv, `external_secrets_username`)
	assert.Contains(t, openbaoEnv, `openbao_address       = "http://127.0.0.1:8200"`)
	assert.Contains(t, openbaoEnv, `"test-cluster-dev-external-secrets"`)
	assert.Contains(t, openbaoEnv, `manage_openbao_kubernetes_auth_backend`)
	assert.Contains(t, openbaoEnv, `manage_external_secrets_kubernetes_auth_role = true`)
	assert.Contains(t, openbaoEnv, `manage_openbao_userpass_auth_backend`)
	assert.Contains(t, openbaoEnv, `manage_external_secrets_user`)
	assert.Contains(t, openbaoEnv, `manage_image_pull_secret             = false`)
	assert.NotContains(t, openbaoEnv, `manage_argocd_customer_repository_credentials`)
	assert.NotContains(t, openbaoEnv, `openbao_token`)
	assert.NotContains(t, openbaoEnv, `image_pull_secret =`)
	assert.Contains(t, setEnvSh, `export TF_VAR_argo_oauth2_client_id=""`)
	assert.Contains(t, setEnvSh, `export TF_VAR_grafana_oauth2_client_secret=""`)
	assert.Contains(t, setEnvSh, `export TF_VAR_oauth2_client_secret=""`)
	assert.Contains(t, setEnvSh, `export TF_VAR_image_pull_secret="e30="`)
	assert.Contains(t, setEnvPs1, `$env:TF_VAR_argo_oauth2_client_id = ""`)
	assert.Contains(t, setEnvPs1, `$env:TF_VAR_oauth2_client_secret = ""`)
	assert.Contains(t, setEnvPs1, `$env:TF_VAR_image_pull_secret = "e30="`)
	assert.NotContains(t, setEnvSh, `argocd_customer_repository`)
	assert.Contains(t, openbaoOutputs, `output "external_secrets_kubernetes_auth_role_name"`)
	assert.Contains(t, openbaoOutputs, `output "openbao_namespace_kv_read_role_name"`)
	assert.Contains(t, openbaoOutputs, `output "external_secrets_password_b64"`)
	assert.Contains(t, openbaoSecretsExample, `manage_t_cloud_public_clouds_yaml = true`)
	assert.NotContains(t, openbaoSecretsExample, `argocd_customer_repository`)
	assert.Contains(t, openbaoSecretsExample, `manage_openbao_oidc_auth_backend = true`)
	assert.Contains(t, openbaoSecretsExample, `https://test.example.com/openbao/ui/vault/auth/oidc/oidc/callback`)
}

func TestTemplateFiles_TCloudPublicEnvAutoTfvarsUsesGenericCCENodePoolFlavor(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var infrastructureEnv string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt" {
			infrastructureEnv = result.Content
		}
	}

	require.NotEmpty(t, infrastructureEnv)
	assert.Contains(t, infrastructureEnv, `flavor             = "s3.xlarge.4"`)
	assert.NotContains(t, infrastructureEnv, `flavor             = "c3.2xl.4"`)
}

func TestTemplateFiles_TCloudPublicProviderOverridesExternalDNSValues(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/t-cloud-public/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "ghcr.io/opentelekomcloud/external-dns-t-cloud-public-webhook")
	assert.Contains(t, externalDNSValues, "tag: 1.1.2")
	assert.Contains(t, externalDNSValues, "secretName: tcloudpubliccloudsyaml")
	assert.Contains(t, externalDNSValues, "key: clouds.yaml")
	assert.NotContains(t, externalDNSValues, "stackit")
}

func TestTemplateFiles_TCloudPublicExternalSecretsValuesConfigureOpenBaoClusterSecretStore(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
				"terraform": map[string]any{
					"provider":       "t-cloud-public",
					"kubernetesType": "cce",
				},
				"services": fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalSecretsValues string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/t-cloud-public/example/external-secrets/values.yaml.tplt" {
			externalSecretsValues = result.Content
		}
	}

	require.NotEmpty(t, externalSecretsValues)
	assert.Contains(t, externalSecretsValues, `name: test-cluster-dev`)
	assert.Contains(t, externalSecretsValues, `server: http://openbao.openbao.svc.cluster.local:8200`)
	assert.Contains(t, externalSecretsValues, `path: secret`)
	assert.Contains(t, externalSecretsValues, `version: v2`)
	assert.Contains(t, externalSecretsValues, `mountPath: k8s-auth`)
	assert.Contains(t, externalSecretsValues, `role: external-secrets`)
	assert.Contains(t, externalSecretsValues, `namespace: external-secrets`)
	assert.NotContains(t, externalSecretsValues, "stackit")
}

func TestTemplateFiles_TCloudPublicCCETraefikValuesRenderELBAnnotations(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"terraform": map[string]any{
					"provider":       "t-cloud-public",
					"kubernetesType": "cce",
				},
				"services": fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var traefikValues string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/example/traefik/values.yaml.tplt" {
			traefikValues = result.Content
		}
	}

	require.NotEmpty(t, traefikValues)
	assert.Contains(t, traefikValues, `kubernetes.io/elb.id: "CHANGE_ME_TERRAFORM_OUTPUT_LOAD_BALANCER_ID"`)
	assert.Contains(t, traefikValues, `kubernetes.io/elb.class: "union"`)
}

func TestTemplateFiles_TCloudPublicCCEHomerDashboardIncludesOpenBaoLink(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"terraform": map[string]any{
					"provider":       "t-cloud-public",
					"kubernetesType": "cce",
				},
				"services": fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var homerValues string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/example/homer-dashboard/values.yaml.tplt" {
			homerValues = result.Content
		}
	}

	require.NotEmpty(t, homerValues)
	assert.Contains(t, homerValues, `name: "openbao"`)
	assert.Contains(t, homerValues, `url: "/openbao/ui/"`)
	assert.Contains(t, homerValues, `logo: "/assets/tools/secretsmanager.png"`)
}

func TestTemplateFiles_StackitProviderOverridesExternalDNSValues(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "stackit",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/stackit/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "ghcr.io/stackitcloud/external-dns-stackit-webhook")
	assert.Contains(t, externalDNSValues, "secretName: external-dns-webhook")
	assert.NotContains(t, externalDNSValues, "opentelekomcloud")
}

func TestTemplateFiles_CommonExternalDNSValuesAreProviderNeutral(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type: Helm,
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "externalSecrets: {}")
	assert.NotContains(t, externalDNSValues, "ghcr.io/stackitcloud")
	assert.NotContains(t, externalDNSValues, "ghcr.io/opentelekomcloud")
	assert.NotContains(t, externalDNSValues, "stackit")
}

func TestTemplateFiles_TCloudPublicExternalDNSSkipsExternalSecretWhenDisabled(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	services := fullServiceContext()
	services["external-secrets"] = map[string]any{"status": "disabled"}

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         services,
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/t-cloud-public/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
		}
	}

	require.NotEmpty(t, externalDNSValues)
	assert.Contains(t, externalDNSValues, "externalSecrets: {}")
	assert.NotContains(t, externalDNSValues, "remoteKey: t-cloud-public-clouds-yaml")
}

func TestTemplateFiles(t *testing.T) {
	tests := []struct {
		name     string
		tplType  TemplateType
		context  map[string]any
		wantErr  bool
		validate func(t *testing.T, results []TemplateResult)
	}{
		{
			name:    "Success: Successfully template all files of type All",
			tplType: All,
			context: map[string]any{
				"var": map[string]any{
					"project_id": "12345",
					"name":       "test-cluster",
					"stage":      "dev",
				},
				"cluster": map[string]any{
					"type":             "hub",
					"name":             "test-cluster",
					"stage":            "dev",
					"dnsName":          "test.example.com",
					"ingressClassName": "traefik",
					"ssoOrg":           "myorg",
					"ssoTeam":          "myteam",
					"terraform": map[string]any{
						"kubernetesType": "ske",
					},
					"argocd": map[string]any{
						"repo": map[string]any{
							"https": map[string]any{
								"managed": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
						"helmRepo": map[string]any{
							"url": "https://charts.example.com",
						},
					},
					"services": fullServiceContext(),
				},
				"catalog": fullCatalogContext(),
			},
			wantErr: false, // No errors expected with valid context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				// Should have both template and static files
				hasTemplate := false
				hasStatic := false
				hasValidTemplate := false

				for _, result := range results {
					if strings.HasSuffix(result.Path, ".tplt") {
						hasTemplate = true
						if result.Error == nil {
							hasValidTemplate = true
							assert.NotEmpty(t, result.Content)
						}
					} else {
						hasStatic = true
						assert.NoError(t, result.Error)
						assert.NotEmpty(t, result.Content)
					}
				}
				assert.True(t, hasTemplate, "Should have at least one template file")
				assert.True(t, hasStatic, "Should have at least one static file")
				assert.True(t, hasValidTemplate, "Should have at least one successfully rendered template")
			},
		},
		{
			name:    "Error: Handle template execution errors in all files",
			tplType: All,
			context: map[string]any{},
			wantErr: true,
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				templateFiles := 0
				staticSuccess := 0
				templateErrors := 0
				for _, result := range results {
					if strings.HasSuffix(result.Path, ".tplt") {
						templateFiles++
						if result.Error != nil {
							templateErrors++
						}
					} else {
						if result.Error == nil {
							staticSuccess++
						}
					}
				}
				assert.Greater(t, templateFiles, 0, "Should have template files")
				assert.Greater(t, templateErrors, 0, "Should have template errors with empty context")
				assert.Greater(t, staticSuccess, 0, "Should have successful static files")
			},
		},
		{
			name:    "Success: Template all Terraform files",
			tplType: Terraform,
			context: map[string]any{
				"var": map[string]any{
					"project_id": "12345",
					"name":       "tf-cluster",
					"stage":      "staging",
				},
				"cluster": map[string]any{
					"terraform": map[string]any{
						"kubernetesType": "ske",
					},
				},
			},
			wantErr: false, // Changed to false with proper context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				for _, result := range results {
					assert.Contains(t, result.Path, "terraform")
					assert.False(t, strings.Contains(result.Path, "helm"), "Should not include helm files")
				}
			},
		},
		{
			name:    "Success: Template all Helm files",
			tplType: Helm,
			context: map[string]any{
				"cluster": map[string]any{
					"type":             "hub",
					"name":             "helm-cluster",
					"stage":            "production",
					"dnsName":          "helm.example.com",
					"ingressClassName": "traefik",
					"ssoOrg":           "myorg",
					"ssoTeam":          "myteam",
					"argocd": map[string]any{
						"repo": map[string]any{
							"https": map[string]any{
								"managed": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
					},
					"services": fullServiceContext(),
				},
				"catalog": fullCatalogContext(),
			},
			wantErr: false, // Changed to false with proper context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				for _, result := range results {
					assert.Contains(t, result.Path, "helm")
					assert.False(t, strings.Contains(result.Path, "terraform"), "Should not include terraform files")
				}
			},
		},
		{
			name:    "Error: Invalid template type",
			tplType: TemplateType(99),
			context: map[string]any{},
			wantErr: true,
			validate: func(t *testing.T, results []TemplateResult) {
				assert.Empty(t, results)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestFS(t)
			defer cleanup()

			results, err := TemplateFiles(TemplateOptions{
				Type: tt.tplType,
				Data: tt.context,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, results)
			}
		})
	}
}
