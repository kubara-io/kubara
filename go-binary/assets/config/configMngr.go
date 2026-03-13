package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/invopop/jsonschema"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	goYaml "gopkg.in/yaml.v3"

	"github.com/knadh/koanf/parsers/yaml"
)

// Manager handles reading and writing configuration
type Manager struct {
	filepath string
	config   *Config
}

func NewConfigManager(filePath string) *Manager {
	return &Manager{
		filepath: filePath,
		config:   &Config{},
	}
}

// Load loads configuration
func (cm *Manager) Load() error {
	k := koanf.New(".")
	// Load from file
	if err := k.Load(file.Provider(cm.filepath), yaml.Parser()); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	// Decoderconfig is used to handle mapping between koanf variables and structs with inline fields when reading from file
	dc := &mapstructure.DecoderConfig{
		TagName:          "yaml",
		WeaklyTypedInput: false,
		Result:           cm.config,
		Squash:           true,
	}

	uc := koanf.UnmarshalConf{
		Tag:           "yaml",
		FlatPaths:     false,
		DecoderConfig: dc,
	}
	if err := k.UnmarshalWithConf("", cm.config, uc); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// GenerateSchema generates a JSON schema from the Config struct
func GenerateSchema() (map[string]any, error) {
	r := jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  false,
	}
	// Build schema from the root using a single reflector
	sch := r.ReflectFromType(reflect.TypeOf(Config{}))

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

	return schemaDoc, nil
}

func (cm *Manager) Validate() error {
	schemaDoc, err := GenerateSchema()
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
		var verr *schemaValidator.ValidationError
		if errors.As(err, &verr) {
			return fmt.Errorf("config validation failed: %v", verr.Causes)
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
	// Ensure directory exists
	filePath := cm.filepath
	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	var b bytes.Buffer
	encoder := goYaml.NewEncoder(&b)
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
