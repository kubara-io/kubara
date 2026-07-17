package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultBootstrapCatalogReference = "oci://ghcr.io/kubara-io/catalogs/bootstrap:1.0.0"

func CreateCatalogFixtures(root string) (string, string, error) {
	bootstrapPath := filepath.Join(root, "bootstrap-catalog")
	generalPath := filepath.Join(root, "general-catalog")

	if err := writeBootstrapCatalog(bootstrapPath); err != nil {
		return "", "", err
	}
	if err := writeGeneralCatalog(generalPath); err != nil {
		return "", "", err
	}

	return bootstrapPath, generalPath, nil
}

func writeBootstrapCatalog(root string) error {
	files := map[string]string{
		"Catalog.yaml":         catalogManifestYAML("bootstrap", "1.0.0"),
		"services/argocd.yaml": serviceDefinitionYAML("argocd", "argo-cd", "disabled", ""),
		"services/crds.yaml":   serviceDefinitionYAML("crds", "crds", "disabled", ""),
		"platform-components/helm/argo-cd/Chart.yaml.tplt":         "{{ if not .cluster }}{{ fail \"missing cluster\" }}{{ end }}\napiVersion: v2\nname: argocd\nversion: 0.1.0\n",
		"platform-components/helm/crds/Chart.yaml.tplt":            "apiVersion: v2\nname: crds\nversion: 0.1.0\n",
		"platform-components/helm/argo-cd/README.md":               "bootstrap chart\n",
		"platform-configs/helm/argo-cd/values.generated.yaml.tplt": "server: {}\n",
		"platform-configs/helm/crds/values.generated.yaml.tplt":    "{}\n",
		"platform-components/terraform/stackit/.keep.tplt":         "# keep\n",
		"platform-configs/terraform/stackit/.keep.tplt":            "# keep\n",
		"platform-components/terraform/images/public-cloud-0.png":  "png\n",
	}

	return writeFixtureFiles(root, files)
}

func writeGeneralCatalog(root string) error {
	files := map[string]string{
		"Catalog.yaml":               catalogManifestYAML("general", "1.0.0"),
		"services/cert-manager.yaml": serviceDefinitionYAML("cert-manager", "cert-manager", "enabled", certManagerSchemaYAML()),
		"platform-components/terraform/stackit/modules/ske-cluster/main.tf":            "module \"ske_cluster\" {}\n",
		"platform-configs/terraform/stackit/infrastructure/main.tf.tplt":               "{{ if not .cluster }}{{ fail \"missing cluster\" }}{{ end }}\nresource \"example\" \"main\" {}\n",
		"platform-configs/terraform/stackit/infrastructure/outputs.tf.tplt":            "output \"name\" { value = \"example\" }\n",
		"platform-configs/terraform/stackit/infrastructure/variables.tf.tplt":          "variable \"name\" { type = string }\n",
		"platform-configs/terraform/stackit/infrastructure/env.auto.tfvars.tplt":       "name = \"example\"\n",
		"platform-components/terraform/t-cloud-public/modules/cce-cluster/main.tf":     "file_permission = \"0600\"\n",
		"platform-components/terraform/t-cloud-public/modules/network/main.tf":         "module \"network\" {}\n",
		"platform-components/terraform/t-cloud-public/modules/storage-classes/main.tf": "module \"storage_classes\" {}\n",
		"platform-configs/terraform/t-cloud-public/infrastructure/main.tf.tplt":        "module \"cluster\" {\n  source = \"../../../../platform-components/terraform/t-cloud-public/modules/cce-cluster\"\n}\n",
		"platform-components/helm/cert-manager/README.md":                              "general chart\n",
	}

	for _, serviceName := range []string{
		"cert-manager",
	} {
		files[filepath.ToSlash(filepath.Join("platform-components", "helm", serviceName, "Chart.yaml.tplt"))] =
			fmt.Sprintf("apiVersion: v2\nname: %s\nversion: 0.1.0\n", serviceName)
	}

	return writeFixtureFiles(root, files)
}

func writeFixtureFiles(root string, files map[string]string) error {
	for relPath, content := range files {
		targetPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create fixture directory for %q: %w", relPath, err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("write fixture file %q: %w", relPath, err)
		}
	}

	return nil
}

func catalogManifestYAML(name, version string) string {
	return fmt.Sprintf(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: %s
spec:
  version: %s
`, name, version)
}

func serviceDefinitionYAML(name, chartPath, status, configSchema string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: %s
spec:
  chartPath: %s
  status: %s
`, name, chartPath, status)

	if strings.TrimSpace(configSchema) != "" {
		b.WriteString(indentYAML("configSchema:\n"+configSchema, 2))
	}

	return b.String()
}

func certManagerSchemaYAML() string {
	return `  type: object
  additionalProperties: false
  properties:
    clusterIssuer:
      type: object
      default: {}
      additionalProperties: false
      properties:
        name:
          type: string
          default: letsencrypt-staging
        email:
          type: string
          default: yourname@your-domain.de
        server:
          type: string
          default: https://acme-staging-v02.api.letsencrypt.org/directory
`
}

func metallbSchemaYAML() string {
	return `  type: object
  additionalProperties: false
  properties:
    publicLoadBalancerIPs:
      type: string
    loadBalancerAddressPool:
      type: array
      items:
        type: string
`
}

func indentYAML(value string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n") + "\n"
}
