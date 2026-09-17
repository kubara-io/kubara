package config

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"path"
	"sync"

	"github.com/kubara-io/libkubara/crdvalidate"
	"github.com/kubara-io/libkubara/manifest"
)

const (
	PlatformSetupAPIVersion = "kubara.io/v1alpha5"
	PlatformSetupKind       = "PlatformSetup"
)

//go:embed crd/kubara.io_platformsetups.yaml
var platformSetupCRD []byte

var (
	validatorOnce sync.Once
	validator     *crdvalidate.Validator
	validatorErr  error
)

func PlatformSetupCRD() []byte {
	return append([]byte(nil), platformSetupCRD...)
}

func configurationValidator() (*crdvalidate.Validator, error) {
	validatorOnce.Do(func() {
		validator, validatorErr = crdvalidate.Compile(bytes.NewReader(platformSetupCRD))
		if validatorErr != nil {
			validatorErr = fmt.Errorf("compile embedded PlatformSetup CRD: %w", validatorErr)
		}
	})
	return validator, validatorErr
}

// ConfigurationJSONSchema returns the selected CRD version's root schema for
// editor validation of PlatformSetup documents.
func ConfigurationJSONSchema() (map[string]any, error) {
	object, err := manifest.DecodeOneBytes(platformSetupCRD)
	if err != nil {
		return nil, fmt.Errorf("decode PlatformSetup CRD: %w", err)
	}
	versions, found, err := object.NestedSlice("spec", "versions")
	if err != nil || !found {
		return nil, fmt.Errorf("PlatformSetup CRD has no versions")
	}
	targetVersion := path.Base(PlatformSetupAPIVersion)
	for _, version := range versions {
		versionMap, ok := version.(map[string]any)
		if !ok || versionMap["name"] != targetVersion {
			continue
		}
		schema, ok := versionMap["schema"].(map[string]any)
		if !ok {
			break
		}
		root, ok := schema["openAPIV3Schema"].(map[string]any)
		if !ok {
			break
		}
		out := make(map[string]any, len(root)+4)
		for key, value := range root {
			out[key] = value
		}
		out["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		out["$id"] = fmt.Sprintf("https://kubara.io/schemas/platformsetup-%s.json", targetVersion)
		out["$defs"] = map[string]any{}
		out["title"] = PlatformSetupKind
		return out, nil
	}
	return nil, fmt.Errorf("PlatformSetup CRD has no %s OpenAPI schema", targetVersion)
}

func decodePlatformSetup(data []byte) (*Config, error) {
	object, err := manifest.DecodeOneBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode PlatformSetup manifest: %w", err)
	}
	v, err := configurationValidator()
	if err != nil {
		return nil, err
	}

	result := v.ValidateCreate(context.Background(), object, crdvalidate.RejectUnknown)
	var setup PlatformSetup
	if err := result.Into(&setup); err != nil {
		return nil, fmt.Errorf("validate PlatformSetup %q: %w", object.Name(), err)
	}
	normalizeDisabledTerraform(&setup.Spec)
	return &setup.Spec, nil
}

// ValidatePlatformSetupTransition validates a proposed CR against its
// previous GitOps version and returns the normalized proposed configuration.
func ValidatePlatformSetupTransition(ctx context.Context, oldData, proposedData []byte) (*Config, error) {
	oldObject, err := manifest.DecodeOneBytes(oldData)
	if err != nil {
		return nil, fmt.Errorf("decode previous PlatformSetup manifest: %w", err)
	}
	proposedObject, err := manifest.DecodeOneBytes(proposedData)
	if err != nil {
		return nil, fmt.Errorf("decode proposed PlatformSetup manifest: %w", err)
	}
	v, err := configurationValidator()
	if err != nil {
		return nil, err
	}
	result := v.ValidateTransition(ctx, proposedObject, oldObject, crdvalidate.RejectUnknown)
	var setup PlatformSetup
	if err := result.Into(&setup); err != nil {
		return nil, fmt.Errorf(
			"validate PlatformSetup transition from %q to %q: %w",
			oldObject.Name(),
			proposedObject.Name(),
			err,
		)
	}
	normalizeDisabledTerraform(&setup.Spec)
	return &setup.Spec, nil
}
