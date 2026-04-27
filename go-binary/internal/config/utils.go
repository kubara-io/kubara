package config

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	schemadefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubara-io/kubara/internal/service"
)

func toMap(schema *apiextensionsv1.JSONSchemaProps) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}

	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(schema)
	if err != nil {
		return nil, fmt.Errorf("convert schema to map: %w", err)
	}
	return out, nil
}

// applySchemaDefaults applies Kubernetes-style schema defaults to an
// unstructured config object using `JSONSchemaProps`.
//
// It converts the public `apiextensions/v1` schema into the internal
// apiextensions type, builds a structural schema, normalizes explicit `null`
// values, and then applies defaults in place onto `obj`.
//
// Semantics:
//
//   - defaults apply only to missing fields
//   - explicit zero values like `0`, `false`, `""`, `[]`, and `{}` are preserved
//   - explicit `null` is normalized before defaulting according to schema
//   - nested defaults only apply if the parent object exists, unless the parent
//     itself has a default such as `{}`
//
// Example schema defaults:
//
//	replicas: 1
//	server: {}
//	server.port: 8080
//
// Input:
//
//	{"name":"demo"}
//
// Outcome:
//
//	{"name":"demo","replicas":1,"server":{"port":8080}}
//
// Notes:
//
// - this function only applies defaults; it does not validate
// - `obj` is mutated in place
func applySchemaDefaults(schema *apiextensionsv1.JSONSchemaProps, obj map[string]any) (service.Config, error) {
	if schema == nil {
		return nil, nil
	}

	if obj == nil {
		obj = map[string]any{}
	}

	// Convert from the public/versioned schema type to the internal schema type
	// expected by the apiserver defaulting/validation internals.
	internal := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.
		Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(
			schema,
			internal,
			nil,
		); err != nil {
		return nil, fmt.Errorf("convert schema for defaulting: %w", err)
	}

	// Build the structural schema representation used by the Kubernetes schema
	// machinery. Defaulting operates on this derived structural form, not
	// directly on JSONSchemaProps.
	structural, err := structuralschema.NewStructural(internal)
	if err != nil {
		return nil, fmt.Errorf("build structural schema: %w", err)
	}

	// Normalize explicit nulls before defaulting.
	// This is required because defaults apply to missing fields, not arbitrary
	// explicit nulls.
	schemadefaulting.PruneNonNullableNullsWithoutDefaults(obj, structural)

	// Apply schema defaults recursively onto the unstructured object.
	// Only missing fields are defaulted.
	schemadefaulting.Default(obj, structural)
	if len(obj) == 0 {
		return nil, nil
	}

	// The object is still unstructured, but now contains schema-applied default
	// values. The surrounding code treats this map as service.Config.
	return service.Config(obj), nil
}
