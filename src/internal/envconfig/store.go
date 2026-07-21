package envconfig

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// EnvStore handles reading and writing configuration
type EnvStore struct {
	K         *koanf.Koanf
	filepath  string
	envMap    *EnvMap
	envPrefix string
}

func NewEnvStore(filePath, delim, envPrfx string) *EnvStore {
	return &EnvStore{
		K:         koanf.New(delim),
		filepath:  filePath,
		envMap:    &EnvMap{},
		envPrefix: envPrfx,
	}
}

func (em *EnvStore) SetDefaults() {
	em.envMap.setDefaults()
}

func (em *EnvStore) GetConfig() *EnvMap {
	return em.envMap
}

func (em *EnvStore) GetFilepath() string {
	return em.filepath
}

// Load loads variables from file and environment
func (em *EnvStore) Load() error {
	// Load from file first (if it exists)
	if _, err := os.Stat(em.filepath); err == nil {
		if err := em.K.Load(file.Provider(em.filepath), dotenv.Parser()); err != nil {
			return fmt.Errorf("load env file: %w", err)
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
		return fmt.Errorf("load environment variables: %w", err)
	}

	// Unmarshal into struct
	var config EnvMap
	if err := em.K.Unmarshal("", &config); err != nil {
		return fmt.Errorf("unmarshal env map: %w", err)
	}

	em.envMap = &config

	return nil
}

func (em *EnvStore) Validate() error {
	err := em.envMap.Validate()
	if err != nil {
		return err
	}
	return nil
}

func (em *EnvStore) GenerateInitialEnvs() ([]byte, error) {
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
			fmt.Fprintf(&b, "%s='%s'\n", koanfKey, defaultVal)
		}
	}

	return []byte(b.String()), nil
}

func (em *EnvStore) GenerateEnvFileFromCurrentValues() ([]byte, error) {
	return RenderEnvFileFromValues(em.GetConfig())
}

func RenderEnvFileFromValues(envMap *EnvMap) ([]byte, error) {
	if envMap == nil {
		return nil, fmt.Errorf("env map is required")
	}

	var b strings.Builder

	v := reflect.ValueOf(envMap).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		if doc := fieldType.Tag.Get("doc"); doc != "" {
			b.WriteString(doc + "\n")
		}

		koanfKey := fieldType.Tag.Get("koanf")
		if koanfKey != "" {
			val := fmt.Sprint(fieldVal.Interface())
			val = strings.ReplaceAll(val, "\\", "\\\\")
			val = strings.ReplaceAll(val, "\"", "\\\"")
			val = strings.ReplaceAll(val, "\n", "\\n")
			fmt.Fprintf(&b, "%s=\"%s\"\n", koanfKey, val)
		}
	}

	return []byte(b.String()), nil
}

// GetCurrentDotEnv returns a new EnvMap for a filepath
// The function looks at the file loads and validates the EnvMap
// Encapsulates loading and validation with EnvMapEnvStore
func GetCurrentDotEnv(filePath string) (EnvMap, error) {
	manager := NewEnvStore(filePath, ".", "")
	if err := manager.Load(); err != nil {
		return EnvMap{}, fmt.Errorf("load env file: %w", err)
	}
	envMap := *manager.GetConfig()
	if err := envMap.Validate(); err != nil {
		return EnvMap{}, fmt.Errorf("validate env map: %w", err)
	}

	return envMap, nil
}
