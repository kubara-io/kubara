package config

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	schemadefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	"k8s.io/apimachinery/pkg/runtime"

	"kubara/assets/service"
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

func applySchemaDefaults(schema *apiextensionsv1.JSONSchemaProps, obj map[string]any) (service.Config, error) {
	if schema == nil {
		return nil, nil
	}

	if obj == nil {
		obj = map[string]any{}
	}

	internal := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(schema, internal, nil); err != nil {
		return nil, fmt.Errorf("convert schema for defaulting: %w", err)
	}

	structural, err := structuralschema.NewStructural(internal)
	if err != nil {
		return nil, fmt.Errorf("build structural schema: %w", err)
	}

	schemadefaulting.Default(obj, structural)
	if len(obj) == 0 {
		return nil, nil
	}

	return service.Config(obj), nil
}
