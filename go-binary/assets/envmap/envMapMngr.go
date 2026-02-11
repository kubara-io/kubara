package envmap

import (
	"fmt"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"os"
	"reflect"
	"strings"
)

// Manager handles reading and writing configuration
type Manager struct {
	K         *koanf.Koanf
	filepath  string
	envMap    *EnvMap
	envPrefix string
}

func NewEnvMapManager(filePath, delim, envPrfx string) *Manager {
	return &Manager{
		K:         koanf.New(delim),
		filepath:  filePath,
		envMap:    &EnvMap{},
		envPrefix: envPrfx,
	}
}

func (em *Manager) SetDefaults() {
	em.envMap.setDefaults()
}

func (em *Manager) GetConfig() *EnvMap {
	return em.envMap
}

func (em *Manager) GetFilepath() string {
	return em.filepath
}

// Load loads variables from file and environment
func (em *Manager) Load() error {
	// Load from file first (if it exists)
	if _, err := os.Stat(em.filepath); err == nil {
		if err := em.K.Load(file.Provider(em.filepath), dotenv.Parser()); err != nil {
			return fmt.Errorf("error loading file: %w", err)
		}
	}

	// Load from environment variables (these will override file values)
	prefix := em.envPrefix
	if err := em.K.Load(env.Provider(".", env.Opt{
		Prefix: prefix,
		TransformFunc: func(k, v string) (string, any) {
			return strings.TrimPrefix(k, prefix), v
		},
		EnvironFunc: nil,
	}), nil); err != nil {
		return fmt.Errorf("error loading environment variables: %w", err)
	}

	// Unmarshal into struct
	var config EnvMap
	if err := em.K.Unmarshal("", &config); err != nil {
		return fmt.Errorf("error unmarshaling envMap: %w", err)
	}

	em.envMap = &config

	return nil
}

func (em *Manager) Validate() error {
	err := em.envMap.Validate()
	if err != nil {
		return err
	}
	return nil
}

func (em *Manager) ValidateAndSaveToFile(path string) error {
	err := em.Validate()
	if err != nil {
		return err
	}
	return em.SaveToFile(path)
}

func (em *Manager) SaveToFile(path string) error {
	var b strings.Builder
	v := reflect.ValueOf(em.envMap).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)
		koanfKey := fieldType.Tag.Get("koanf")
		if koanfKey == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s='%v'\n", koanfKey, fieldVal.Interface()))
	}

	return os.WriteFile(path, []byte(b.String()), 0600)
}

func (em *Manager) GenerateEnvExample() ([]byte, error) {
	var b strings.Builder

	envMap := em.GetConfig() // Use the existing config for default values
	envMap.setDefaults()

	v := reflect.ValueOf(envMap).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)

		// Handle documentation/comment tags
		if doc := fieldType.Tag.Get("doc"); doc != "" {
			b.WriteString(doc + "\n")
		}

		// Handle env var fields
		koanfKey := fieldType.Tag.Get("koanf")
		if koanfKey != "" {
			// Use the default value from the tag, or the current value
			defaultVal := fieldType.Tag.Get("default")
			b.WriteString(fmt.Sprintf("%s='%s'\n", koanfKey, defaultVal))
		}
	}

	return []byte(b.String()), nil
}
