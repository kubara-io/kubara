package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
	"github.com/rs/zerolog/log"
)

// applyConfigDefaults defaults routing settings and validates their combinations.
func applyConfigDefaults(cfg *Config) error {
	for i := range cfg.Clusters {
		cluster := &cfg.Clusters[i]
		if cluster.Networking == nil {
			cluster.Networking = &ClusterNetworking{}
		}
		networking := cluster.Networking
		if cluster.IngressClassName != "" {
			log.Warn().Str("cluster", cluster.Name).Msg("ingressClassName is deprecated; use networking.ingressClassName")
			if networking.Gateway == nil {
				if networking.IngressClassName != "" && networking.IngressClassName != cluster.IngressClassName {
					return fmt.Errorf("cluster %q: ingressClassName conflicts with networking.ingressClassName", cluster.Name)
				}
				networking.IngressClassName = cluster.IngressClassName
			}
		}
		if networking.Gateway != nil && networking.IngressClassName != "" {
			return fmt.Errorf("cluster %q: networking cannot contain both ingressClassName and gateway", cluster.Name)
		}
		if networking.Gateway == nil && networking.IngressClassName == "" {
			networking.IngressClassName = "traefik"
		}
		if networking.Gateway == nil {
			cluster.IngressClassName = networking.IngressClassName
		} else if cluster.IngressClassName == "" {
			cluster.IngressClassName = "traefik"
		}
		for name, svc := range cluster.Services {
			if svc.Networking != nil && svc.Networking.Gateway != nil && networking.Gateway == nil {
				return fmt.Errorf("cluster %q service %q: a service Gateway override requires networking.gateway on the cluster", cluster.Name, name)
			}
		}
	}
	applyDefaults(cfg)
	return nil
}

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
	for v.Kind() == reflect.Pointer {
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
	for part := range strings.SplitSeq(tag, ",") {
		if strings.HasPrefix(part, "default=") {
			return part[len("default="):], true
		}
	}
	return "", false
}
