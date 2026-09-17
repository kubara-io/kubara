package catalog

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/kubara-io/libkubara/crdvalidate"
	"github.com/kubara-io/libkubara/manifest"
)

//go:embed crd/kubara.io_catalogs.yaml
var catalogCRD []byte

//go:embed crd/kubara.io_servicedefinitions.yaml
var serviceDefCRD []byte

var catalogValidator = sync.OnceValues(func() (*crdvalidate.Validator, error) {
	validator, err := crdvalidate.Compile(bytes.NewReader(catalogCRD))
	if err != nil {
		return nil, fmt.Errorf("compile embedded Catalog CRD: %w", err)
	}
	return validator, nil
})

var serviceDefValidator = sync.OnceValues(func() (*crdvalidate.Validator, error) {
	validator, err := crdvalidate.Compile(bytes.NewReader(serviceDefCRD))
	if err != nil {
		return nil, fmt.Errorf("compile embedded ServiceDefinition CRD: %w", err)
	}
	return validator, nil
})

// DecodeCatalogManifest decodes and validates a Catalog manifest from raw YAML bytes.
func DecodeCatalogManifest(data []byte) (CatalogManifest, error) {
	obj, err := manifest.DecodeOneBytes(data)
	if err != nil {
		return CatalogManifest{}, fmt.Errorf("decode Catalog manifest: %w", err)
	}
	validator, err := catalogValidator()
	if err != nil {
		return CatalogManifest{}, err
	}
	res := validator.ValidateCreate(context.Background(), obj, crdvalidate.RejectUnknown)
	if err := res.Err(); err != nil {
		return CatalogManifest{}, fmt.Errorf("validate Catalog %q: %w", obj.Name(), err)
	}
	var manifest CatalogManifest
	if err := res.Into(&manifest); err != nil {
		return CatalogManifest{}, fmt.Errorf("unmarshal Catalog %q: %w", obj.Name(), err)
	}
	return manifest, nil
}

// DecodeServiceDefinition decodes and validates a ServiceDefinition manifest from raw YAML bytes.
func DecodeServiceDefinition(data []byte) (ServiceDefinition, error) {
	obj, err := manifest.DecodeOneBytes(data)
	if err != nil {
		return ServiceDefinition{}, fmt.Errorf("decode ServiceDefinition manifest: %w", err)
	}
	validator, err := serviceDefValidator()
	if err != nil {
		return ServiceDefinition{}, err
	}
	res := validator.ValidateCreate(context.Background(), obj, crdvalidate.RejectUnknown)
	if err := res.Err(); err != nil {
		return ServiceDefinition{}, fmt.Errorf("validate ServiceDefinition %q: %w", obj.Name(), err)
	}
	var svcDef ServiceDefinition
	if err := res.Into(&svcDef); err != nil {
		return ServiceDefinition{}, fmt.Errorf("unmarshal ServiceDefinition %q: %w", obj.Name(), err)
	}
	return svcDef, nil
}
