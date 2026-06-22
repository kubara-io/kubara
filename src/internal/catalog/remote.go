package catalog

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	defaultCredentialsRel = ".kubara/credentials.json"
	tempLayoutTag         = "cached"
)

type LoginOptions struct {
	Registry      string
	Username      string
	Password      string
	IdentityToken string
	Insecure      bool
}

type LoginResult struct {
	Registry        string
	CredentialsPath string
}

type PullOptions struct {
	Reference string
	Insecure  bool
}

type PullResult struct {
	Artifact  CachedArtifact
	Reference string
	Updated   bool
}

type PushOptions struct {
	Reference string
	From      string
	Insecure  bool
}

type PushResult struct {
	Artifact   CachedArtifact
	Reference  string
	Uploaded   bool
	SourceFrom string
}

func LoginRegistry(ctx context.Context, options LoginOptions) (LoginResult, error) {
	if err := options.Validate(); err != nil {
		return LoginResult{}, err
	}

	store, credentialsPath, err := newKubaraCredentialStore()
	if err != nil {
		return LoginResult{}, err
	}

	registryName := strings.TrimSpace(options.Registry)
	registryClient, err := remote.NewRegistry(registryName)
	if err != nil {
		return LoginResult{}, fmt.Errorf("parse registry %q: %w", registryName, err)
	}
	registryClient.Client = newRegistryAuthClient(store, options.Insecure)

	credential := auth.Credential{
		Username:     strings.TrimSpace(options.Username),
		Password:     strings.TrimSpace(options.Password),
		RefreshToken: strings.TrimSpace(options.IdentityToken),
	}
	if err := credentials.Login(ctx, store, registryClient, credential); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Registry:        registryName,
		CredentialsPath: credentialsPath,
	}, nil
}

func (options LoginOptions) Validate() error {
	registry := strings.TrimSpace(options.Registry)
	username := strings.TrimSpace(options.Username)
	identityToken := strings.TrimSpace(options.IdentityToken)

	if registry == "" {
		return fmt.Errorf("registry is required")
	}

	if identityToken != "" {
		if username != "" || options.Password != "" {
			return fmt.Errorf("identity token authentication cannot be combined with username or password")
		}
		return nil
	}

	if username == "" {
		return fmt.Errorf("username is required when not using an identity token")
	}
	if options.Password == "" {
		return fmt.Errorf("password is required when not using an identity token")
	}

	return nil
}

func PullCatalog(ctx context.Context, options PullOptions) (PullResult, error) {
	ref, err := ParseOCIReference(options.Reference)
	if err != nil {
		return PullResult{}, err
	}

	cachedArtifact, _ := GetCachedArtifact(ref.Raw)
	artifact, err := pullRemoteCatalog(ctx, ref, options.Insecure)
	if err != nil {
		return PullResult{}, err
	}
	if !ref.IsDigest {
		if err := writeLocalReference(ref, artifact); err != nil {
			return PullResult{}, err
		}
	}

	return PullResult{
		Artifact:  artifact,
		Reference: ref.Raw,
		Updated:   cachedArtifact.ManifestDigest != artifact.ManifestDigest,
	}, nil
}

func PushCatalog(ctx context.Context, options PushOptions) (PushResult, error) {
	ref, err := ParseOCIReference(options.Reference)
	if err != nil {
		return PushResult{}, err
	}
	if ref.IsDigest {
		return PushResult{}, fmt.Errorf("push destination %q must use a tag, not a digest", ref.Raw)
	}

	artifact, sourceFrom, err := resolveCachedPushSource(options)
	if err != nil {
		return PushResult{}, err
	}

	if ref.Tag != artifact.CatalogVersion {
		return PushResult{}, fmt.Errorf(
			"destination tag %q must match catalog version %q",
			ref.Tag,
			artifact.CatalogVersion,
		)
	}

	if err := pushCachedArtifact(ctx, ref, artifact, options.Insecure); err != nil {
		return PushResult{}, err
	}
	if err := writeLocalReference(ref, artifact); err != nil {
		return PushResult{}, err
	}

	return PushResult{
		Artifact:   artifact,
		Reference:  ref.Raw,
		Uploaded:   true,
		SourceFrom: sourceFrom,
	}, nil
}

func resolveCachedPushSource(options PushOptions) (CachedArtifact, string, error) {
	sourceFrom := strings.TrimSpace(options.From)
	if sourceFrom != "" {
		artifact, err := GetCachedArtifact(sourceFrom)
		if err != nil {
			return CachedArtifact{}, "", err
		}
		return artifact, sourceFrom, nil
	}

	artifact, err := GetCachedArtifact(options.Reference)
	if err != nil {
		return CachedArtifact{}, "", err
	}
	return artifact, options.Reference, nil
}

func pushCachedArtifact(ctx context.Context, ref OCIReference, artifact CachedArtifact, insecure bool) error {
	repo, err := newRemoteRepository(ref, insecure)
	if err != nil {
		return err
	}

	artifactDir, err := artifactDirPath(artifact.ManifestDigest)
	if err != nil {
		return err
	}
	layoutStore, err := oci.New(filepath.Join(artifactDir, "layout"))
	if err != nil {
		return fmt.Errorf("open cached OCI layout: %w", err)
	}

	if _, err := oras.Copy(
		ctx,
		layoutStore,
		artifact.ManifestDigest,
		repo,
		ref.Reference,
		oras.DefaultCopyOptions,
	); err != nil {
		return fmt.Errorf("push catalog %q: %w", ref.Raw, err)
	}
	return nil
}

func pullRemoteCatalog(ctx context.Context, ref OCIReference, insecure bool) (CachedArtifact, error) {
	tempArtifactDir, cleanup, err := newTempArtifactDir()
	if err != nil {
		return CachedArtifact{}, err
	}
	defer cleanup()

	repo, err := newRemoteRepository(ref, insecure)
	if err != nil {
		return CachedArtifact{}, err
	}

	layoutStore, err := oci.New(filepath.Join(tempArtifactDir, "layout"))
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("create OCI layout store: %w", err)
	}

	desc, err := oras.Copy(
		ctx,
		repo,
		ref.Reference,
		layoutStore,
		tempLayoutTag,
		oras.DefaultCopyOptions,
	)
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("pull catalog %q: %w", ref.Raw, err)
	}

	rootDir, err := extractCatalogContents(
		ctx,
		layoutStore,
		desc,
		filepath.Join(tempArtifactDir, "contents"),
	)
	if err != nil {
		return CachedArtifact{}, err
	}

	manifest, err := LoadCatalogManifest(rootDir)
	if err != nil {
		return CachedArtifact{}, err
	}

	artifact := CachedArtifact{
		SchemaVersion:  cacheSchemaVersion,
		CatalogName:    manifest.Metadata.Name,
		CatalogVersion: manifest.Spec.Version,
		ManifestDigest: desc.Digest.String(),
		RootDirectory:  filepath.Base(rootDir),
	}
	if err := finalizeCachedArtifact(tempArtifactDir, artifact); err != nil {
		return CachedArtifact{}, err
	}

	return artifact, nil
}

func newRemoteRepository(ref OCIReference, insecure bool) (*remote.Repository, error) {
	store, _, err := newKubaraCredentialStore()
	if err != nil {
		return nil, err
	}

	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", ref.Registry, ref.Repository))
	if err != nil {
		return nil, fmt.Errorf("create remote repository %q: %w", ref.Raw, err)
	}
	repo.Client = newRegistryAuthClient(store, insecure)

	return repo, nil
}

func newRegistryAuthClient(store credentials.Store, insecure bool) *auth.Client {
	client := *auth.DefaultClient
	client.Client = newRegistryHTTPClient(insecure)
	client.Cache = auth.NewCache()
	client.Credential = credentials.Credential(store)
	return &client
}

func newRegistryHTTPClient(insecure bool) *http.Client {
	if !insecure {
		return retry.DefaultClient
	}

	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 - this is only used if the user explicitely sets the insecure flag
	}

	return &http.Client{
		Transport: retry.NewTransport(baseTransport),
	}
}

func newKubaraCredentialStore() (credentials.Store, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("resolve user home directory: %w", err)
	}

	path := filepath.Join(home, defaultCredentialsRel)
	store, err := credentials.NewFileStore(path)
	if err != nil {
		return nil, "", fmt.Errorf("open kubara registry credentials store %q: %w", path, err)
	}
	return store, path, nil
}
