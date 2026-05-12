package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/go-viper/mapstructure/v2"
	"github.com/invopop/jsonschema"
	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// ConfigStore handles reading and writing configuration
type ConfigStore struct {
	filepath       string
	config         *Config
	catalog        *catalog.Catalog
	catalogOptions catalog.LoadOptions
}

func NewConfigStore(filePath string) *ConfigStore {
	return NewConfigStoreWithCatalog(filePath, catalog.LoadOptions{})
}

func NewConfigStoreWithCatalog(filePath string, catalogOptions catalog.LoadOptions) *ConfigStore {
	return &ConfigStore{
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
	if err := cs.applyServiceCatalogDefaults(); err != nil {
		return fmt.Errorf("apply service catalog defaults: %w", err)
	}

	return nil
}

// GenerateSchema generates a JSON schema from the Config struct
func GenerateSchema() (map[string]any, error) {
	return GenerateSchemaWithCatalog(catalog.LoadOptions{})
}

// GenerateSchemaWithCatalog generates a JSON schema from the Config struct
// with optional external service definitions merged into the built-in catalog.
func GenerateSchemaWithCatalog(catalogOptions catalog.LoadOptions) (map[string]any, error) {
	cat, err := catalog.Load(catalogOptions)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}

	return generateSchemaWithCatalog(cat)
}

func generateSchemaWithCatalog(cat catalog.Catalog) (map[string]any, error) {
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
	if err := composeServiceSchema(schemaDoc, cat); err != nil {
		return nil, fmt.Errorf("compose service schema: %w", err)
	}

	return schemaDoc, nil
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

func (cs *ConfigStore) Validate() error {
	cat, err := cs.GetCatalog()
	if err != nil {
		return fmt.Errorf("load catalog: %w", err)
	}

	schemaDoc, err := generateSchemaWithCatalog(cat)
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
	return nil

}

// GetConfig returns the current configuration struct.
func (cs *ConfigStore) GetConfig() *Config {
	return cs.config
}

// GetCatalog returns the catalog for this config store, loading it on first use.
func (cs *ConfigStore) GetCatalog() (catalog.Catalog, error) {
	if cs.catalog != nil {
		return *cs.catalog, nil
	}

	cat, err := catalog.Load(cs.catalogOptions)
	if err != nil {
		return catalog.Catalog{}, fmt.Errorf("load catalog: %w", err)
	}

	cs.catalog = &cat
	return *cs.catalog, nil
}

// GetFilepath returns the filepath for the config.
func (cs *ConfigStore) GetFilepath() string {
	return cs.filepath
}

// SaveToFile saves the configuration to a YAML file
func (cs *ConfigStore) SaveToFile() error {
	if strings.TrimSpace(cs.config.Version) == "" {
		cs.config.Version = ConfigVersionV1Alpha1
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
	required := make([]any, 0, len(keys))
	for _, serviceName := range keys {
		definition := cat.Services[serviceName]
		instanceSchema, err := buildServiceInstanceSchema(definition)
		if err != nil {
			return nil, fmt.Errorf("build schema for service %q: %w", serviceName, err)
		}
		serviceProperties[serviceName] = instanceSchema
		required = append(required, serviceName)
	}

	return map[string]any{
		"type":                 "object",
		"title":                "Services",
		"description":          "Configuration for deployed services.",
		"additionalProperties": false,
		"properties":           serviceProperties,
		"required":             required,
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

func (cs *ConfigStore) applyServiceCatalogDefaults() error {
	cat, err := cs.GetCatalog()
	if err != nil {
		return err
	}

	for i, cluster := range cs.config.Clusters {
		if cluster.Services == nil {
			cluster.Services = make(service.Services, len(cat.Services))
		}

		for name, def := range cat.Services {
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
