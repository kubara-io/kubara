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

	"kubara/catalog"

	"github.com/go-viper/mapstructure/v2"
	"github.com/invopop/jsonschema"
	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// Manager handles reading and writing configuration
type Manager struct {
	filepath       string
	config         *Config
	catalogOptions catalog.LoadOptions
}

func NewConfigManager(filePath string) *Manager {
	return NewConfigManagerWithCatalog(filePath, catalog.LoadOptions{})
}

func NewConfigManagerWithCatalog(filePath string, catalogOptions catalog.LoadOptions) *Manager {
	return &Manager{
		filepath:       filePath,
		config:         &Config{},
		catalogOptions: catalogOptions,
	}
}

// Load loads configuration
func (cm *Manager) Load() error {
	data, err := os.ReadFile(cm.filepath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse yaml config: %w", err)
	}
	migrated, err := migrateRawConfig(raw)
	if err != nil {
		return err
	}

	dc := &mapstructure.DecoderConfig{
		TagName:          "yaml",
		WeaklyTypedInput: false,
		Result:           cm.config,
		Squash:           true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(decodeServiceHook()),
	}
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return fmt.Errorf("failed to initialize config decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	applyDefaults(cm.config)
	if err := applyServiceCatalogDefaults(cm.config, cm.catalogOptions); err != nil {
		return err
	}
	if migrated {
		if err := cm.SaveToFile(); err != nil {
			return fmt.Errorf("failed to persist migrated config: %w", err)
		}
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

	cat, err := catalog.Load(catalogOptions)
	if err != nil {
		return nil, fmt.Errorf("failed loading catalog: %w", err)
	}
	if err := composeServiceSchema(schemaDoc, cat); err != nil {
		return nil, err
	}

	return schemaDoc, nil
}

func (cm *Manager) Validate() error {
	if err := applyServiceCatalogDefaults(cm.config, cm.catalogOptions); err != nil {
		return err
	}

	schemaDoc, err := GenerateSchemaWithCatalog(cm.catalogOptions)
	if err != nil {
		return err
	}

	const schemaURL = "mem://config.schema.json"
	c := schemaValidator.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	// Validate instance by value
	var instance any
	data, err := json.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	if err := compiled.Validate(instance); err != nil {
		if verr, ok := errors.AsType[*schemaValidator.ValidationError](err); ok {
			return fmt.Errorf("config validation errors: %v", verr.Causes)
		}
		return fmt.Errorf("config not valid: %w", err)
	}
	return nil

}

// GetConfig returns the current configuration struct.
func (cm *Manager) GetConfig() *Config {
	return cm.config
}

// GetFilepath returns the filepath for the config.
func (cm *Manager) GetFilepath() string {
	return cm.filepath
}

// SaveToFile saves the configuration to a YAML file
func (cm *Manager) SaveToFile() error {
	if strings.TrimSpace(cm.config.Version) == "" {
		cm.config.Version = ConfigVersionV1Alpha1
	}

	// Ensure directory exists
	filePath := cm.filepath
	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err := encoder.Encode(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, b.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
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
			return nil, fmt.Errorf("failed to build schema for service %q: %w", serviceName, err)
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
			"enum":        []any{string(StatusEnabled), string(StatusDisabled)},
		},
	}

	if definition.Spec.ConfigSchema != nil {
		configSchema, err := catalog.ToMap(definition.Spec.ConfigSchema)
		if err != nil {
			return nil, err
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
