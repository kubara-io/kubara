package config

import (
	"reflect"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

// applyDefaults walks the config struct tree via reflection and sets zero-value
// fields to their default, as declared in the `jsonschema:"default=..."` tag.
// This keeps the jsonschema struct tags as the single source of truth for both
// schema generation (invopop/jsonschema) and runtime defaulting.
func applyDefaults(v any) {
	applyDefaultsValue(reflect.ValueOf(v))
}

func applyDefaultsValue(v reflect.Value) {
	// Deference pointers to get to the underlying value
	// If we encounter a nil pointer, we can't set defaults on it, so we just return.
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		applyDefaultsStruct(v)
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			applyDefaultsValue(v.Index(i))
		}
	}
}

func applyDefaultsStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		// Skip unexported fields
		if !fv.CanSet() {
			continue
		}

		// For embedded/anonymous structs, recurse into them directly.
		if field.Anonymous {
			applyDefaultsValue(fv)
			continue
		}

		// Apply default if the field is a settable zero value.
		if def, ok := parseDefaultFromTag(field.Tag.Get("jsonschema")); ok {
			if utils.IsZeroValue(fv) {
				utils.SetFieldValue(fv, def)
			}
		}

		// Recurse into nested structs, pointers, and slices.
		applyDefaultsValue(fv)
	}
}

// parseDefaultFromTag extracts the value from a `default=X` directive inside a
// comma-separated jsonschema tag. It splits on the first `=` so that values
// containing `=` (e.g. URLs) are preserved correctly.
func parseDefaultFromTag(tag string) (string, bool) {
	for _, part := range strings.Split(tag, ",") {
		if strings.HasPrefix(part, "default=") {
			return part[len("default="):], true
		}
	}
	return "", false
}
