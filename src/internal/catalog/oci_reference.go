package catalog

import (
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
	"oras.land/oras-go/v2/registry"
)

const (
	CatalogArtifactType    = "application/vnd.kubara.catalog.v1"
	CatalogLayerMediaType  = "application/vnd.kubara.catalog.layer.v1.tar+gzip"
	cacheSchemaVersion     = "v1"
	defaultCatalogCacheRel = ".kubara/catalogs"
	defaultLocalCatalogRef = "oci://localhost/"
	ociScheme              = "oci://"
)

type OCIReference struct {
	Raw        string
	Registry   string
	Repository string
	Reference  string
	Tag        string
	Digest     digest.Digest
	IsDigest   bool
}

type CachedArtifact struct {
	SchemaVersion  string `json:"schemaVersion"`
	CatalogName    string `json:"catalogName"`
	CatalogVersion string `json:"catalogVersion"`
	ManifestDigest string `json:"manifestDigest"`
	RootDirectory  string `json:"rootDirectory"`
}

type cachedReference struct {
	SchemaVersion  string `json:"schemaVersion"`
	ManifestDigest string `json:"manifestDigest"`
	CatalogName    string `json:"catalogName"`
	CatalogVersion string `json:"catalogVersion"`
	Reference      string `json:"reference,omitempty"`
	Registry       string `json:"registry,omitempty"`
	Repository     string `json:"repository,omitempty"`
	Tag            string `json:"tag,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}

func IsOCIReference(value string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), ociScheme)
}

func GetCachedArtifact(reference string) (CachedArtifact, error) {
	ref, err := ParseOCIReference(reference)
	if err != nil {
		return CachedArtifact{}, err
	}

	if ref.IsDigest {
		if artifact, found, err := findArtifactByDigest(ref.Digest); err != nil {
			return CachedArtifact{}, err
		} else if found {
			return artifact, nil
		}
		return CachedArtifact{}, fmt.Errorf("digest does not exist locally")
	}

	if artifact, found, err := findTagArtifact(ref); err != nil {
		return CachedArtifact{}, err
	} else if found {
		return artifact, nil
	}

	if isLocalhostReference(ref) {
		return CachedArtifact{}, fmt.Errorf("cached local catalog %q was not found; package it first with `kubara catalog package`", ref.Raw)
	}

	return CachedArtifact{}, fmt.Errorf("cached catalog %q was not found", ref.Raw)
}

func ParseOCIReference(raw string) (OCIReference, error) {
	trimmed := strings.TrimSpace(raw)
	if !IsOCIReference(trimmed) {
		return OCIReference{}, fmt.Errorf("catalog reference %q must use the oci:// scheme", raw)
	}

	parsed, err := registry.ParseReference(strings.TrimPrefix(trimmed, ociScheme))
	if err != nil {
		return OCIReference{}, fmt.Errorf("parse OCI reference %q: %w", raw, err)
	}
	if strings.TrimSpace(parsed.Reference) == "" {
		return OCIReference{}, fmt.Errorf("OCI reference %q must include either a tag or a digest", raw)
	}

	if err := parsed.ValidateReferenceAsDigest(); err == nil {
		dgst, err := parsed.Digest()
		if err != nil {
			return OCIReference{}, fmt.Errorf("parse digest from %q: %w", raw, err)
		}
		return OCIReference{
			Raw:        trimmed,
			Registry:   parsed.Registry,
			Repository: parsed.Repository,
			Reference:  parsed.Reference,
			Digest:     dgst,
			IsDigest:   true,
		}, nil
	}

	if err := parsed.ValidateReferenceAsTag(); err != nil {
		return OCIReference{}, fmt.Errorf("invalid OCI tag in %q: %w", raw, err)
	}
	if !StrictCatalogVersion.MatchString(parsed.Reference) {
		return OCIReference{}, fmt.Errorf(`OCI tag %q must match exact semantic version format "x.y.z" without a leading "v"`, parsed.Reference)
	}

	return OCIReference{
		Raw:        trimmed,
		Registry:   parsed.Registry,
		Repository: parsed.Repository,
		Reference:  parsed.Reference,
		Tag:        parsed.Reference,
		IsDigest:   false,
	}, nil
}

func ParseOCIReferenceBase(raw string) (OCIReference, error) {
	base := strings.TrimSpace(raw)
	if base == "" {
		base = defaultLocalCatalogRef
	}
	if !IsOCIReference(base) {
		return OCIReference{}, fmt.Errorf("catalog reference base %q must use the oci:// scheme", raw)
	}

	normalized := base
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}

	ociRef := strings.TrimSuffix(strings.TrimPrefix(normalized, ociScheme), "/")
	if parsedBase, err := registry.ParseReference(ociRef); err == nil && strings.TrimSpace(parsedBase.Reference) != "" {
		return OCIReference{}, fmt.Errorf("OCI reference base %q must not include a tag or digest", base)
	}

	placeholder := "kubara-placeholder"
	validationRef := ociRef + "/" + placeholder + ":0.0.1"
	parsed, err := registry.ParseReference(validationRef)
	if err != nil {
		return OCIReference{}, fmt.Errorf("parse OCI reference base %q: %w", base, err)
	}

	return OCIReference{
		Raw:        normalized,
		Registry:   parsed.Registry,
		Repository: strings.TrimSuffix(strings.TrimSuffix(parsed.Repository, placeholder), "/"),
	}, nil
}

func BuildCatalogReference(catalogName, version, rawBase string) (OCIReference, error) {
	base, err := ParseOCIReferenceBase(rawBase)
	if err != nil {
		return OCIReference{}, err
	}

	repository := catalogName
	if base.Repository != "" {
		repository = fmt.Sprintf("%s/%s", base.Repository, catalogName)
	}
	return ParseOCIReference(fmt.Sprintf("oci://%s/%s:%s", base.Registry, repository, version))
}
