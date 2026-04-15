package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"kubara/assets/catalog"

	"github.com/go-viper/mapstructure/v2"
	"go.yaml.in/yaml/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func createServicesFromCatalogWithOptions(options catalog.LoadOptions, clusterType string) (Services, error) {
	cat, err := catalog.Load(options)
	if err != nil {
		return nil, err
	}
	return serviceDefaultsFromCatalog(cat, clusterType), nil
}

func decodeServiceHook() mapstructure.DecodeHookFuncType {
	return func(_ reflect.Type, to reflect.Type, source any) (any, error) {
		if to != reflect.TypeFor[ServiceInstance]() {
			return source, nil
		}
		return decodeServiceInstance(source)
	}
}

func applyServiceCatalogDefaults(cfg *Config, options catalog.LoadOptions) error {
	cat, err := catalog.Load(options)
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}

	for i := range cfg.Clusters {
		defaults := serviceDefaultsFromCatalog(cat, cfg.Clusters[i].Type)
		normalized := normalizeServiceKeys(cfg.Clusters[i].Services)
		if normalized == nil {
			normalized = Services{}
		}

		for serviceName, defaultInstance := range defaults {
			current, exists := normalized[serviceName]
			if !exists {
				normalized[serviceName] = cloneServiceInstance(defaultInstance)
				continue
			}
			if current.Status == "" {
				current.Status = defaultInstance.Status
			}
			current.Config = mergeConfigDefaults(defaultInstance.Config, current.Config)
			normalized[serviceName] = current
		}

		// Unknown services should still default status to disabled if not explicitly set.
		for name, instance := range normalized {
			if instance.Status == "" {
				instance.Status = StatusDisabled
				normalized[name] = instance
			}
		}

		cfg.Clusters[i].Services = normalized
	}

	return nil
}

func normalizeServiceKeys(input Services) Services {
	if input == nil {
		return nil
	}

	normalized := make(Services, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		canonical := catalog.CanonicalServiceName(key)
		if canonical == "" {
			continue
		}

		// Canonical key wins when both canonical and legacy alias are present.
		if key != canonical {
			if _, canonicalExists := input[canonical]; canonicalExists {
				continue
			}
		}

		instance := input[key]
		if existing, ok := normalized[canonical]; ok {
			normalized[canonical] = mergeServiceInstances(existing, instance)
			continue
		}
		normalized[canonical] = instance
	}

	return normalized
}

func mergeServiceInstances(base, override ServiceInstance) ServiceInstance {
	out := cloneServiceInstance(base)
	if override.Status != "" {
		out.Status = override.Status
	}
	out.Config = mergeConfigDefaults(out.Config, override.Config)
	return out
}

func cloneServiceInstance(in ServiceInstance) ServiceInstance {
	return ServiceInstance{
		Status: in.Status,
		Config: cloneMap(in.Config),
	}
}

func serviceDefaultsFromCatalog(cat catalog.Catalog, clusterType string) Services {
	out := make(Services, len(cat.Services))
	for name, def := range cat.Services {
		status := def.Spec.Status
		if !serviceAppliesToClusterType(def, clusterType) {
			status = catalog.StatusDisabled
		}

		instance := ServiceInstance{
			Status: toConfigStatus(status),
			Config: extractDefaultsFromSchema(def.Spec.ConfigSchema),
		}
		out[name] = instance
	}
	return out
}

func toConfigStatus(s catalog.Status) Status {
	switch s {
	case catalog.StatusEnabled:
		return StatusEnabled
	default:
		return StatusDisabled
	}
}

func extractDefaultsFromSchema(schema *apiextensionsv1.JSONSchemaProps) map[string]any {
	if schema == nil {
		return nil
	}

	out := map[string]any{}
	for propertyName, propertySchema := range schema.Properties {
		value, ok := extractDefaultValue(&propertySchema)
		if !ok {
			continue
		}
		out[propertyName] = value
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func extractDefaultValue(schema *apiextensionsv1.JSONSchemaProps) (any, bool) {
	if schema == nil {
		return nil, false
	}

	if schema.Default != nil && len(schema.Default.Raw) > 0 {
		var out any
		if err := json.Unmarshal(schema.Default.Raw, &out); err == nil {
			return out, true
		}
	}

	if len(schema.Properties) == 0 {
		return nil, false
	}

	nested := extractDefaultsFromSchema(schema)
	if len(nested) == 0 {
		return nil, false
	}
	return nested, true
}

func mergeConfigDefaults(defaults, provided map[string]any) map[string]any {
	if len(defaults) == 0 && len(provided) == 0 {
		return nil
	}

	out := cloneMap(defaults)
	for k, v := range provided {
		existing, hasExisting := out[k]
		defaultMap, defaultIsMap := existing.(map[string]any)
		providedMap, providedIsMap := v.(map[string]any)
		if hasExisting && defaultIsMap && providedIsMap {
			out[k] = mergeConfigDefaults(defaultMap, providedMap)
			continue
		}
		out[k] = v
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if nested, ok := v.(map[string]any); ok {
			out[k] = cloneMap(nested)
			continue
		}
		out[k] = v
	}
	return out
}

func (s *ServiceInstance) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	instance, err := decodeServiceInstance(raw)
	if err != nil {
		return err
	}
	*s = instance
	return nil
}

func serviceAppliesToClusterType(def catalog.ServiceDefinition, clusterType string) bool {
	if len(def.Spec.ClusterTypes) == 0 {
		return true
	}

	switch clusterType {
	case "controlplane", "worker":
		for _, allowed := range def.Spec.ClusterTypes {
			if allowed == "*" || allowed == clusterType {
				return true
			}
		}
		return false
	default:
		// Unknown/placeholder cluster types keep the legacy behavior:
		// expose all services with their definition defaults.
		return true
	}
}

func decodeServiceInstance(source any) (ServiceInstance, error) {
	raw, err := toStringMap(source)
	if err != nil {
		return ServiceInstance{}, err
	}

	var out ServiceInstance

	if statusRaw, ok := raw["status"]; ok {
		status, ok := statusRaw.(string)
		if !ok {
			return ServiceInstance{}, fmt.Errorf("service status must be string")
		}
		out.Status = Status(status)
	}

	if configRaw, ok := raw["config"]; ok {
		cfg, err := toStringMap(configRaw)
		if err != nil {
			return ServiceInstance{}, fmt.Errorf("service config must be an object: %w", err)
		}
		out.Config = cfg
	}

	legacyInline := map[string]any{}
	for key, val := range raw {
		if key == "status" || key == "config" {
			continue
		}
		legacyInline[key] = val
	}
	if len(legacyInline) > 0 {
		// Explicit "config" key should win over legacy inline keys when both exist.
		out.Config = mergeConfigDefaults(legacyInline, out.Config)
	}

	return out, nil
}
