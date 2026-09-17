package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config/migrations"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/go-viper/mapstructure/v2"
	"go.yaml.in/yaml/v3"
)

// ConfigStore handles reading and writing configuration
type ConfigStore struct {
	cwd            string
	filepath       string
	config         *Config
	catalogOptions catalog.LoadOptions
}

func NewConfigStore(cwd string, filePath string, catalogOptions catalog.LoadOptions) *ConfigStore {
	if catalogOptions.CWD == "" {
		catalogOptions.CWD = cwd
	}
	return &ConfigStore{
		cwd:            cwd,
		filepath:       filePath,
		config:         &Config{},
		catalogOptions: catalogOptions,
	}
}

// Load loads configuration
func (cs *ConfigStore) Load() error {
	data, err := os.ReadFile(cs.filepath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse YAML config: %w", err)
	}
	if raw["apiVersion"] == PlatformSetupAPIVersion && raw["kind"] == PlatformSetupKind {
		cfg, err := decodePlatformSetup(data)
		if err != nil {
			return err
		}
		cs.config = cfg
		if err := cs.ApplyServiceCatalogDefaults(); err != nil {
			return fmt.Errorf("apply service catalog defaults: %w", err)
		}
		return nil
	}
	if err := cs.loadLegacy(raw); err != nil {
		return err
	}
	document, err := platformSetupDocument(cs.config, "platform")
	if err != nil {
		return fmt.Errorf("convert legacy config to PlatformSetup: %w", err)
	}
	normalized, err := decodePlatformSetup(document)
	if err != nil {
		return fmt.Errorf("validate migrated PlatformSetup: %w", err)
	}
	if err := os.WriteFile(cs.filepath, document, 0600); err != nil {
		return fmt.Errorf("persist migrated PlatformSetup: %w", err)
	}
	cs.config = normalized
	if err := cs.ApplyServiceCatalogDefaults(); err != nil {
		return fmt.Errorf("apply service catalog defaults: %w", err)
	}
	return nil
}

func (cs *ConfigStore) loadLegacy(raw map[string]any) error {
	_, err := migrations.Apply(cs.cwd, raw)
	if err != nil {
		return fmt.Errorf("migration of config failed: %w", err)
	}
	applyLegacyDefaults(raw)

	dc := &mapstructure.DecoderConfig{
		TagName:          "json",
		WeaklyTypedInput: false,
		Result:           cs.config,
		Squash:           true,
	}
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return fmt.Errorf("initialize config decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	normalizeDisabledTerraform(cs.config)
	stripBootstrapServices(cs.config)
	ensureClusterServices(cs.config)
	return nil
}

func ensureClusterServices(cfg *Config) {
	for i := range cfg.Clusters {
		if cfg.Clusters[i].Services == nil {
			cfg.Clusters[i].Services = service.Services{}
		}
	}
}

func normalizeDisabledTerraform(cfg *Config) {
	for index := range cfg.Clusters {
		if terraform := cfg.Clusters[index].Terraform; terraform != nil && terraform.Provider == TerraformProviderNone {
			cfg.Clusters[index].Terraform = nil
		}
	}
}

func stripBootstrapServices(cfg *Config) {
	for i := range cfg.Clusters {
		for name := range cfg.Clusters[i].Services {
			if catalog.IsBootstrapService(name) {
				delete(cfg.Clusters[i].Services, name)
			}
		}
	}
}

// GetConfig returns the current configuration struct.
func (cs *ConfigStore) GetConfig() *Config {
	return cs.config
}

// DynamicJSONSchema returns the generic CRD schema with per-cluster catalog
// service contracts for editor use. It is not used for runtime validation.
func (cs *ConfigStore) DynamicJSONSchema() (map[string]any, error) {
	schema, err := ConfigurationJSONSchema()
	if err != nil {
		return nil, err
	}
	if len(cs.config.Clusters) == 0 {
		return schema, nil
	}
	spec := schema["properties"].(map[string]any)["spec"].(map[string]any)
	clusters := spec["properties"].(map[string]any)["clusters"].(map[string]any)
	item := clusters["items"].(map[string]any)
	branches := make([]any, 0, len(cs.config.Clusters))
	for _, cluster := range cs.config.Clusters {
		encoded, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("clone cluster schema: %w", err)
		}
		var branch map[string]any
		if err := json.Unmarshal(encoded, &branch); err != nil {
			return nil, fmt.Errorf("decode cluster schema: %w", err)
		}
		cat, err := cs.GetCatalogForCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("load catalog for cluster %q: %w", cluster.Name, err)
		}
		services, err := buildServicesSchema(cat.UserConfigurableServices())
		if err != nil {
			return nil, fmt.Errorf("build services schema for cluster %q: %w", cluster.Name, err)
		}
		properties := branch["properties"].(map[string]any)
		properties["name"].(map[string]any)["const"] = cluster.Name
		properties["services"] = services
		branches = append(branches, branch)
	}
	clusters["items"] = map[string]any{"oneOf": branches}
	return schema, nil
}

// GetCatalogForCluster returns the effective catalog for one cluster using the
// shared per-cluster precedence rules.
func (cs *ConfigStore) GetCatalogForCluster(cluster Cluster) (catalog.Catalog, error) {
	loadOptions := CatalogLoadOptions(cs.config, cluster, cs.catalogOptions)

	cat, err := catalog.Load(loadOptions)
	if err != nil {
		return catalog.Catalog{}, fmt.Errorf("load catalog: %w", err)
	}

	return cat, nil
}

// GetFilepath returns the filepath for the config.
func (cs *ConfigStore) GetFilepath() string {
	return cs.filepath
}

// SaveToFile saves the configuration to a YAML file
func (cs *ConfigStore) SaveToFile() error {
	document, err := platformSetupDocument(cs.config, "platform")
	if err != nil {
		return fmt.Errorf("convert config to PlatformSetup: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cs.filepath), 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(cs.filepath, document, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func buildServicesSchema(cat catalog.Catalog) (map[string]any, error) {
	keys := make([]string, 0, len(cat.Services))
	for name := range cat.Services {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	serviceProperties := make(map[string]any, len(keys))
	for _, serviceName := range keys {
		definition := cat.Services[serviceName]
		instanceSchema, err := buildServiceInstanceSchema(definition)
		if err != nil {
			return nil, fmt.Errorf("build schema for service %q: %w", serviceName, err)
		}
		serviceProperties[serviceName] = instanceSchema
	}

	return map[string]any{
		"type":                 "object",
		"title":                "Services",
		"description":          "Configuration for deployed services.",
		"additionalProperties": false,
		"properties":           serviceProperties,
	}, nil
}

func buildServiceInstanceSchema(definition catalog.ServiceDefinition) (map[string]any, error) {
	properties := map[string]any{
		"status": map[string]any{
			"type":        "string",
			"title":       "Service Status",
			"description": "The desired status of the service.",
			"enum":        []any{string(service.StatusEnabled), string(service.StatusDisabled)},
			"default":     string(definition.Spec.Status),
		},
		"storage":    buildServiceStorageSchema(),
		"networking": buildServiceNetworkingSchema(),
	}

	if definition.Spec.ConfigSchema != nil {
		configSchema, err := toMap(definition.Spec.ConfigSchema)
		if err != nil {
			return nil, fmt.Errorf("convert service config schema to map: %w", err)
		}
		defaultConfig, err := applySchemaDefaults(definition.Spec.ConfigSchema, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("apply service config defaults: %w", err)
		}
		configSchema["default"] = defaultConfig
		properties["config"] = configSchema
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             []any{"status"},
	}, nil
}

func buildServiceStorageSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"className": map[string]any{
				"type":        "string",
				"title":       "Storage Class Name",
				"description": "Optional storage class name override for persistent volumes.",
				"minLength":   1,
			},
		},
	}
}

func buildServiceNetworkingSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"annotations": map[string]any{
				"type":                 "object",
				"title":                "Ingress Annotations",
				"description":          "Optional ingress annotation overrides for this service.",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
	}
}

func (cs *ConfigStore) ApplyServiceCatalogDefaults() error {
	for i := range cs.config.Clusters {
		cat, err := cs.GetCatalogForCluster(cs.config.Clusters[i])
		if err != nil {
			return fmt.Errorf("load catalog for cluster %q: %w", cs.config.Clusters[i].Name, err)
		}
		cat = cat.UserConfigurableServices()

		normalizedServices, err := normalizeServiceNames(cs.config.Clusters[i].Services)
		if err != nil {
			return fmt.Errorf("normalize services for cluster %q: %w", cs.config.Clusters[i].Name, err)
		}
		cs.config.Clusters[i].Services = normalizedServices

		for name := range cs.config.Clusters[i].Services {
			if catalog.IsBootstrapService(name) {
				delete(cs.config.Clusters[i].Services, name)
			}
		}

		cluster := cs.config.Clusters[i]
		if cluster.Services == nil {
			cluster.Services = make(service.Services, len(cat.Services))
		}

		for name, def := range cat.Services {
			if len(def.Spec.ClusterTypes) > 0 && !slices.Contains(def.Spec.ClusterTypes, cluster.Type) {
				continue
			}

			existing, exists := cluster.Services[name]
			if !exists {
				cfg, err := applySchemaDefaults(def.Spec.ConfigSchema, map[string]any{})
				if err != nil {
					return fmt.Errorf("apply defaults for service %q: %w", name, err)
				}

				cluster.Services[name] = service.Service{
					Status: def.Spec.Status,
					Config: cfg,
				}
				continue
			}

			statusUpdated := false
			if existing.Status == "" {
				existing.Status = def.Spec.Status
				statusUpdated = true
			}

			if def.Spec.ConfigSchema == nil {
				if statusUpdated {
					cluster.Services[name] = existing
				}
				continue
			}

			base := map[string]any{}
			for k, v := range existing.Config {
				base[k] = service.CloneValue(v)
			}

			cfg, err := applySchemaDefaults(def.Spec.ConfigSchema, base)
			if err != nil {
				return fmt.Errorf("apply defaults for service %q: %w", name, err)
			}

			existing.Config = cfg
			cluster.Services[name] = existing
		}

		cs.config.Clusters[i] = cluster
	}

	return nil
}

func normalizeServiceNames(services service.Services) (service.Services, error) {
	if services == nil {
		return nil, nil
	}

	normalized := make(service.Services, len(services))
	sourceByCanonical := make(map[string]string, len(services))

	for originalName, cfg := range services {
		canonicalName := catalog.CanonicalServiceName(originalName)
		if previousName, exists := sourceByCanonical[canonicalName]; exists {
			return nil, fmt.Errorf("services has conflicting keys %q and %q for canonical service %q", previousName, originalName, canonicalName)
		}

		normalized[canonicalName] = cfg
		sourceByCanonical[canonicalName] = originalName
	}

	return normalized, nil
}
