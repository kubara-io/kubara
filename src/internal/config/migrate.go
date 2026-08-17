package config

import (
	"fmt"
	"slices"

	"sigs.k8s.io/yaml"
)

func applyLegacyDefaults(raw map[string]any) {
	clusters, _ := raw["clusters"].([]any)
	for _, value := range clusters {
		cluster, ok := value.(map[string]any)
		if !ok {
			continue
		}
		setDefault(cluster, "stage", "dev")
		setDefault(cluster, "type", Hub)

		networking, ok := cluster["networking"].(map[string]any)
		if !ok {
			cluster["networking"] = map[string]any{
				"type": "ingress",
				"ingress": map[string]any{
					"className": "traefik",
				},
			}
		} else {
			setDefault(networking, "type", "ingress")
			if networking["type"] == "ingress" {
				ingress, ok := networking["ingress"].(map[string]any)
				if !ok {
					networking["ingress"] = map[string]any{"className": "traefik"}
				} else {
					setDefault(ingress, "className", "traefik")
				}
			}
		}

		if terraform, ok := cluster["terraform"].(map[string]any); ok {
			setDefault(terraform, "provider", string(TerraformProviderNone))
			setDefault(terraform, "kubernetesType", "ske")
		}
		argocd, ok := cluster["argocd"].(map[string]any)
		if !ok {
			continue
		}
		setDefault(argocd, "selfManaged", string(ArgoCDSelfManagedEnabled))
		repo, ok := argocd["repo"].(map[string]any)
		if !ok {
			continue
		}
		for _, protocol := range []string{"https", "oci"} {
			repositoryType, ok := repo[protocol].(map[string]any)
			if !ok {
				continue
			}
			for _, name := range []string{"configs", "components"} {
				repository, ok := repositoryType[name].(map[string]any)
				if ok {
					setDefault(repository, "path", "")
					setDefault(repository, "targetRevision", "main")
				}
			}
		}
	}
}

func setDefault(object map[string]any, key string, value any) {
	current, exists := object[key]
	if !exists || current == nil || current == "" {
		object[key] = value
	}
}

func platformSetupDocument(cfg *Config, name string) ([]byte, error) {
	if name == "" {
		name = "platform"
	}
	cleanCfg := *cfg
	cleanCfg.Clusters = slices.Clone(cfg.Clusters)
	for i := range cleanCfg.Clusters {
		cleanCfg.Clusters[i].IngressClassName = ""
	}

	document := map[string]any{
		"apiVersion": PlatformSetupAPIVersion,
		"kind":       PlatformSetupKind,
		"metadata":   map[string]any{"name": name},
		"spec":       &cleanCfg,
	}
	out, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal PlatformSetup: %w", err)
	}
	return out, nil
}
