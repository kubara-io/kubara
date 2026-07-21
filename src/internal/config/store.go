package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config/migrations"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/go-viper/mapstructure/v2"
	"github.com/invopop/jsonschema"
	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// ConfigStore handles reading and writing configuration
type ConfigStore struct {
	cwd            string
	filepath       string
	config         *Config
	catalogCache   map[string]catalog.Catalog
	catalogOptions catalog.LoadOptions
}

func NewConfigStore(cwd string, filePath string, catalogOptions catalog.LoadOptions) *ConfigStore {
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

	migrated, err := migrations.Apply(cs.cwd, raw)
	if err != nil {
		return fmt.Errorf("migration of config failed: %w", err)
	}

	dc := &mapstructure.DecoderConfig{
		TagName:          "yaml",
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

	applyDefaults(cs.config)
	normalizeDisabledTerraform(cs.config)
	if err := cs.ApplyServiceCatalogDefaults(); err != nil {
		return fmt.Errorf("apply service catalog defaults: %w", err)
	}

	if err = cs.validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	if migrated {
		if err := cs.SaveToFile(); err != nil {
			return fmt.Errorf("persist migrated config: %w", err)
		}
	}

	return nil
}

// GenerateSchema generates a JSON schema from the Config struct using the
// same per-cluster catalog resolution as config validation and generation.
func (cs *ConfigStore) GenerateSchema() (map[string]any, error) {
	if len(cs.config.Clusters) == 0 {
		return generateSchemaWithCatalog(catalog.Catalog{Services: map[string]catalog.ServiceDefinition{}})
	}

	clusterCatalogs := make([]catalog.Catalog, 0, len(cs.config.Clusters))
	unionCatalog := catalog.Catalog{Services: map[string]catalog.ServiceDefinition{}}
	for _, cluster := range cs.config.Clusters {
		cat, err := cs.GetCatalogForCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("load catalog for cluster %q: %w", cluster.Name, err)
		}
		clusterCatalogs = append(clusterCatalogs, cat)
		for name, def := range cat.Services {
			if _, exists := unionCatalog.Services[name]; !exists {
				unionCatalog.Services[name] = def
			}
		}
	}

	schemaDoc, err := generateSchemaWithCatalog(unionCatalog)
	if err != nil {
		return nil, err
	}
	if err := applyClusterSpecificSchemaBranches(schemaDoc, cs.config.Clusters, clusterCatalogs); err != nil {
		return nil, err
	}

	return schemaDoc, nil
}

func generateSchemaWithCatalog(cat catalog.Catalog) (map[string]any, error) {
	cat = cat.UserConfigurableServices()

	r := jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  false,
	}
	// Build schema from the root using a single reflector
	sch := r.ReflectFromType(reflect.TypeFor[Config]())

	const schemaURL = "mem://config.schema.json"
	if sch.ID == "" {
		sch.ID = schemaURL
	}

	// Marshal to bytes then decode into map[string]any
	b, err := json.Marshal(sch)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	var schemaDoc map[string]any
	if err := json.Unmarshal(b, &schemaDoc); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}
	ensureServiceConfigDefinition(schemaDoc)
	allowDisabledTerraformSchema(schemaDoc)
	if err := composeServiceSchema(schemaDoc, cat); err != nil {
		return nil, fmt.Errorf("compose service schema: %w", err)
	}

	return schemaDoc, nil
}

func allowDisabledTerraformSchema(schemaDoc map[string]any) {
	defs, ok := schemaDoc["$defs"].(map[string]any)
	if !ok {
		return
	}

	terraform, ok := defs["Terraform"].(map[string]any)
	if !ok {
		return
	}

	required, ok := terraform["required"].([]any)
	if !ok || len(required) == 0 {
		return
	}

	terraform["anyOf"] = []any{
		map[string]any{
			"properties": map[string]any{
				"provider": map[string]any{
					"const": string(TerraformProviderNone),
				},
			},
		},
		map[string]any{
			"required": required,
		},
	}
	delete(terraform, "required")
}

func normalizeDisabledTerraform(cfg *Config) {
	for idx := range cfg.Clusters {
		terraform := cfg.Clusters[idx].Terraform
		if terraform != nil && terraform.Provider == TerraformProviderNone {
			cfg.Clusters[idx].Terraform = nil
		}
	}
}

// ensureServiceConfigDefinition ensures that for every service the
// config schema document object is properly generated even
func ensureServiceConfigDefinition(schemaDoc map[string]any) {
	defs, ok := schemaDoc["$defs"].(map[string]any)
	if !ok {
		return
	}
	if _, exists := defs["Config"]; exists {
		return
	}
	defs["Config"] = map[string]any{
		"type":                 "object",
		"title":                "Service Config",
		"description":          "Service-specific configuration",
		"additionalProperties": true,
	}
}

func (cs *ConfigStore) validate() error {
	schemaDoc, err := cs.GenerateSchema()
	if err != nil {
		return fmt.Errorf("generate schema: %w", err)
	}

	const schemaURL = "mem://config.schema.json"
	c := schemaValidator.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	// Validate instance by value
	var instance any
	data, err := json.Marshal(cs.config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	if err := compiled.Validate(instance); err != nil {
		if verr, ok := errors.AsType[*schemaValidator.ValidationError](err); ok {
			return fmt.Errorf("validate config: %w", verr)
		}
		return fmt.Errorf("validate config: %w", err)
	}
	if err := validateProviderKubernetesTypes(cs.config); err != nil {
		return fmt.Errorf("validate provider kubernetes types: %w", err)
	}
	return nil
}

func applyClusterSpecificSchemaBranches(schemaDoc map[string]any, clusters []Cluster, catalogs []catalog.Catalog) error {
	defs, ok := schemaDoc["$defs"].(map[string]any)
	if !ok {
		return fmt.Errorf("catalog schema is missing $defs")
	}

	clusterDef, ok := defs["Cluster"].(map[string]any)
	if !ok || len(clusters) == 0 {
		return nil
	}

	branches := make([]any, 0, len(clusters))
	for i, cluster := range clusters {
		branch, err := deepCopySchemaMap(clusterDef)
		if err != nil {
			return fmt.Errorf("clone cluster schema for %q: %w", cluster.Name, err)
		}

		properties, ok := branch["properties"].(map[string]any)
		if !ok {
			return fmt.Errorf("cluster schema is missing properties")
		}

		nameSchema, ok := properties["name"].(map[string]any)
		if !ok {
			return fmt.Errorf("cluster schema is missing name property")
		}
		nameBranch, err := deepCopySchemaMap(nameSchema)
		if err != nil {
			return fmt.Errorf("clone name schema for %q: %w", cluster.Name, err)
		}
		nameBranch["const"] = cluster.Name
		properties["name"] = nameBranch

		clusterServices := catalogs[i].UserConfigurableServices()
		servicesSchema, err := buildServicesSchema(clusterServices)
		if err != nil {
			return fmt.Errorf("build service schema for cluster %q: %w", cluster.Name, err)
		}
		properties["services"] = servicesSchema

		branches = append(branches, branch)
	}

	defs["Cluster"] = map[string]any{
		"oneOf": branches,
	}
	return nil
}

func deepCopySchemaMap(input map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}

	return output, nil
}

func validateProviderKubernetesTypes(cfg *Config) error {
	for _, cluster := range cfg.Clusters {
		if cluster.Terraform == nil {
			continue
		}

		provider := cluster.Terraform.Provider
		kubernetesType := cluster.Terraform.KubernetesType
		supportedTypes := supportedKubernetesTypesForProvider(provider)
		if len(supportedTypes) == 0 {
			continue
		}
		if slices.Contains(supportedTypes, kubernetesType) {
			continue
		}

		return fmt.Errorf("cluster %q uses terraform.provider %q with terraform.kubernetesType %q; supported kubernetes types for %q are: %s",
			cluster.Name,
			provider,
			kubernetesType,
			provider,
			strings.Join(supportedTypes, ", "),
		)
	}

	return nil
}

func supportedKubernetesTypesForProvider(provider TerraformProvider) []string {
	switch provider {
	case TerraformProviderStackit:
		return []string{"ske", "edge"}
	case TerraformProviderTCloudPublic:
		return []string{"cce"}
	default:
		return nil
	}
}

// GetConfig returns the current configuration struct.
func (cs *ConfigStore) GetConfig() *Config {
	return cs.config
}

// GetCatalogForCluster returns the effective catalog for one cluster using the
// shared per-cluster precedence rules.
func (cs *ConfigStore) GetCatalogForCluster(cluster Cluster) (catalog.Catalog, error) {
	loadOptions := EffectiveCatalogLoadOptionsForCluster(cs.config, cluster, cs.catalogOptions)
	cacheKey := catalogCacheKey(loadOptions)
	if cs.catalogCache != nil {
		if cat, exists := cs.catalogCache[cacheKey]; exists {
			return cat, nil
		}
	}

	cat, err := catalog.Load(loadOptions)
	if err != nil {
		return catalog.Catalog{}, fmt.Errorf("load catalog: %w", err)
	}

	if cs.catalogCache == nil {
		cs.catalogCache = make(map[string]catalog.Catalog)
	}
	cs.catalogCache[cacheKey] = cat

	return cat, nil
}

// GetFilepath returns the filepath for the config.
func (cs *ConfigStore) GetFilepath() string {
	return cs.filepath
}

// SaveToFile saves the configuration to a YAML file
func (cs *ConfigStore) SaveToFile() error {
	if strings.TrimSpace(cs.config.Version) == "" {
		cs.config.Version = ConfigVersionV1Alpha4
	}

	// Ensure directory exists
	filePath := cs.filepath
	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Marshal to YAML
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err := encoder.Encode(cs.config)
	if err != nil {
		return fmt.Errorf("marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, b.Bytes(), 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func composeServiceSchema(schemaDoc map[string]any, cat catalog.Catalog) error {
	defs, ok := schemaDoc["$defs"].(map[string]any)
	if !ok {
		return fmt.Errorf("catalog schema is missing $defs")
	}

	servicesSchema, err := buildServicesSchema(cat)
	if err != nil {
		return err
	}
	defs["Services"] = servicesSchema
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
		},
		"storage":    buildServiceStorageSchema(),
		"networking": buildServiceNetworkingSchema(),
	}

	if definition.Spec.ConfigSchema != nil {
		configSchema, err := toMap(definition.Spec.ConfigSchema)
		if err != nil {
			return nil, fmt.Errorf("convert service config schema to map: %w", err)
		}
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
