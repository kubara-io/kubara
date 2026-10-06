package helm

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/rs/zerolog/log"
	release "helm.sh/helm/v4/pkg/release/v1"
	releaseutil "helm.sh/helm/v4/pkg/release/v1/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const argoHookAnnotation = "argocd.argoproj.io/hook"

type Hooks interface {
	PreApply(context.Context, []*release.Hook) error
	PostApply(context.Context, []*release.Hook) error
	PreDelete(context.Context, []*release.Hook) error
	PostDelete(context.Context, []*release.Hook) error

	DeleteByPolicy(
		context.Context,
		*release.Hook,
		release.HookDeletePolicy,
	) error
}

type hookClient interface {
	ApplyManifest(
		context.Context,
		[]byte,
		k8s.ApplyOptions,
	) error

	DeleteObjectAndWait(
		context.Context,
		*unstructured.Unstructured,
		time.Duration,
	) error

	WaitForObjectCompletion(
		context.Context,
		*unstructured.Unstructured,
		time.Duration,
	) error
}

type BootstrapHookOptions struct {
	Timeout      time.Duration
	ApplyOptions k8s.ApplyOptions
}

type bootstrapHooks struct {
	client hookClient
	opts   BootstrapHookOptions
}

type ManifestSet struct {
	Hooks     []*release.Hook
	Resources []byte
}

type ManifestAction func(
	context.Context,
	[]byte,
) error

type ReadyAction func(
	context.Context,
) error

var supportedHookEvents = map[string]struct{}{
	release.HookPreInstall.String():  {},
	release.HookPreUpgrade.String():  {},
	release.HookPostInstall.String(): {},
	release.HookPostUpgrade.String(): {},
	release.HookPreDelete.String():   {},
	release.HookPostDelete.String():  {},
}

var supportedDeletePolicies = map[string]struct{}{
	release.HookBeforeHookCreation.String(): {},
	release.HookSucceeded.String():          {},
	release.HookFailed.String():             {},
}

func NewBootstrapHooks(
	client hookClient,
	opts BootstrapHookOptions,
) Hooks {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}

	return &bootstrapHooks{
		client: client,
		opts:   opts,
	}
}

// ParseManifestSet separates Helm hooks from normal resources while preserving
// the original ordering of normal resources.
func ParseManifestSet(
	manifest []byte,
) (ManifestSet, error) {
	documents := releaseutil.SplitManifests(
		string(manifest),
	)

	keys := make([]string, 0, len(documents))
	for key := range documents {
		keys = append(keys, key)
	}

	sort.Sort(
		releaseutil.BySplitManifestsOrder(keys),
	)

	var (
		result    ManifestSet
		resources strings.Builder
	)

	for _, key := range keys {
		content := strings.TrimSpace(
			documents[key],
		)
		if content == "" {
			continue
		}

		var head releaseutil.SimpleHead
		if err := yaml.Unmarshal(
			[]byte(content),
			&head,
		); err != nil {
			return ManifestSet{}, fmt.Errorf(
				"parse manifest %q: %w",
				key,
				err,
			)
		}

		if head.Metadata != nil {
			annotations := head.Metadata.Annotations

			if annotations[release.HookAnnotation] != "" &&
				annotations[argoHookAnnotation] != "" {
				return ManifestSet{}, fmt.Errorf(
					"manifest %q declares both Helm and Argo CD hook annotations",
					key,
				)
			}

			if annotations[release.HookAnnotation] != "" {
				if err := validateHookAnnotations(
					annotations,
				); err != nil {
					return ManifestSet{}, fmt.Errorf(
						"manifest %q: %w",
						key,
						err,
					)
				}
			}
		}

		hooks, generic, err := releaseutil.SortManifests(
			map[string]string{
				key: content,
			},
			nil,
			releaseutil.InstallOrder,
		)
		if err != nil {
			return ManifestSet{}, fmt.Errorf(
				"classify Helm manifest %q: %w",
				key,
				err,
			)
		}

		if len(hooks) > 0 {
			result.Hooks = append(
				result.Hooks,
				hooks...,
			)
			continue
		}

		if len(generic) == 0 {
			return ManifestSet{}, fmt.Errorf(
				"manifest %q was neither a normal resource nor a supported Helm hook",
				key,
			)
		}

		appendManifest(
			&resources,
			content,
		)
	}

	result.Resources = []byte(
		resources.String(),
	)

	return result, nil
}

func ApplyWithHooks(
	ctx context.Context,
	manifest []byte,
	hooks Hooks,
	apply ManifestAction,
	ready ReadyAction,
) error {
	set, err := ParseManifestSet(manifest)
	if err != nil {
		return err
	}

	if err := hooks.PreApply(
		ctx,
		set.Hooks,
	); err != nil {
		return fmt.Errorf(
			"pre-apply hooks: %w",
			err,
		)
	}

	if len(set.Resources) > 0 {
		if err := apply(
			ctx,
			set.Resources,
		); err != nil {
			return fmt.Errorf(
				"apply resources: %w",
				err,
			)
		}
	}

	if ready != nil {
		if err := ready(ctx); err != nil {
			return fmt.Errorf(
				"wait for applied resources: %w",
				err,
			)
		}
	}

	if err := hooks.PostApply(
		ctx,
		set.Hooks,
	); err != nil {
		return fmt.Errorf(
			"post-apply hooks: %w",
			err,
		)
	}

	return nil
}

// DeleteWithHooks provides the matching delete-side lifecycle.
//
// deleteResources must not return until the ordinary resources have actually
// been deleted; PostDelete runs only after that callback succeeds.
func DeleteWithHooks(
	ctx context.Context,
	manifest []byte,
	hooks Hooks,
	deleteResources ManifestAction,
) error {
	set, err := ParseManifestSet(manifest)
	if err != nil {
		return err
	}

	if err := hooks.PreDelete(
		ctx,
		set.Hooks,
	); err != nil {
		return fmt.Errorf(
			"pre-delete hooks: %w",
			err,
		)
	}

	if len(set.Resources) > 0 {
		if err := deleteResources(
			ctx,
			set.Resources,
		); err != nil {
			return fmt.Errorf(
				"delete resources: %w",
				err,
			)
		}
	}

	if err := hooks.PostDelete(
		ctx,
		set.Hooks,
	); err != nil {
		return fmt.Errorf(
			"post-delete hooks: %w",
			err,
		)
	}

	return nil
}

func (h *bootstrapHooks) PreApply(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPreInstall,
		release.HookPreUpgrade,
	)
}

func (h *bootstrapHooks) PostApply(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPostInstall,
		release.HookPostUpgrade,
	)
}

func (h *bootstrapHooks) PreDelete(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPreDelete,
	)
}

func (h *bootstrapHooks) PostDelete(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPostDelete,
	)
}

func (h *bootstrapHooks) run(
	ctx context.Context,
	allHooks []*release.Hook,
	events ...release.HookEvent,
) error {
	executing := hooksForEvents(
		allHooks,
		events...,
	)
	if len(executing) == 0 {
		return nil
	}

	sort.SliceStable(
		executing,
		func(i, j int) bool {
			if executing[i].Weight == executing[j].Weight {
				return executing[i].Name < executing[j].Name
			}

			return executing[i].Weight < executing[j].Weight
		},
	)

	successful := make(
		[]*release.Hook,
		0,
		len(executing),
	)

	for _, hook := range executing {
		if err := h.DeleteByPolicy(
			ctx,
			hook,
			release.HookBeforeHookCreation,
		); err != nil {
			return fmt.Errorf(
				"delete hook %q before creation: %w",
				hook.Name,
				err,
			)
		}

		obj, err := HookObject(hook)
		if err != nil {
			return err
		}

		log.Info().
			Str("kind", obj.GetKind()).
			Str("namespace", obj.GetNamespace()).
			Str("name", obj.GetName()).
			Msg("Executing Helm bootstrap hook")

		if err := h.client.ApplyManifest(
			ctx,
			[]byte(hook.Manifest),
			h.opts.ApplyOptions,
		); err != nil {
			return h.failHook(
				ctx,
				hook,
				successful,
				fmt.Errorf(
					"apply hook %q: %w",
					hook.Name,
					err,
				),
			)
		}

		if err := h.client.WaitForObjectCompletion(
			ctx,
			obj,
			h.opts.Timeout,
		); err != nil {
			return h.failHook(
				ctx,
				hook,
				successful,
				fmt.Errorf(
					"execute hook %q: %w",
					hook.Name,
					err,
				),
			)
		}

		successful = append(
			successful,
			hook,
		)
	}

	var cleanupErrors []error

	for i := len(successful) - 1; i >= 0; i-- {
		if err := h.DeleteByPolicy(
			ctx,
			successful[i],
			release.HookSucceeded,
		); err != nil {
			cleanupErrors = append(
				cleanupErrors,
				fmt.Errorf(
					"delete successful hook %q: %w",
					successful[i].Name,
					err,
				),
			)
		}
	}

	return errors.Join(cleanupErrors...)
}

func (h *bootstrapHooks) failHook(
	ctx context.Context,
	failed *release.Hook,
	successful []*release.Hook,
	hookErr error,
) error {
	errs := []error{
		hookErr,
	}

	if err := h.DeleteByPolicy(
		ctx,
		failed,
		release.HookFailed,
	); err != nil {
		errs = append(
			errs,
			fmt.Errorf(
				"delete failed hook %q: %w",
				failed.Name,
				err,
			),
		)
	}

	for i := len(successful) - 1; i >= 0; i-- {
		if err := h.DeleteByPolicy(
			ctx,
			successful[i],
			release.HookSucceeded,
		); err != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"cleanup successful hook %q: %w",
					successful[i].Name,
					err,
				),
			)
		}
	}

	return errors.Join(errs...)
}

func (h *bootstrapHooks) DeleteByPolicy(
	ctx context.Context,
	hook *release.Hook,
	policy release.HookDeletePolicy,
) error {
	if !hookHasDeletePolicy(
		hook,
		policy,
	) {
		return nil
	}

	obj, err := HookObject(hook)
	if err != nil {
		return err
	}

	if !hookDeletionAllowed(obj) {
		log.Warn().
			Str("kind", obj.GetKind()).
			Str("name", obj.GetName()).
			Msg("Skipping deletion of protected Helm bootstrap hook")

		return nil
	}

	log.Info().
		Str("policy", policy.String()).
		Str("kind", obj.GetKind()).
		Str("namespace", obj.GetNamespace()).
		Str("name", obj.GetName()).
		Msg("Deleting Helm bootstrap hook")

	return h.client.DeleteObjectAndWait(
		ctx,
		obj,
		h.opts.Timeout,
	)
}

func hookHasDeletePolicy(
	hook *release.Hook,
	policy release.HookDeletePolicy,
) bool {
	// Helm defaults an unspecified policy to before-hook-creation.
	if len(hook.DeletePolicies) == 0 {
		return policy == release.HookBeforeHookCreation
	}

	return slices.Contains(
		hook.DeletePolicies,
		policy,
	)
}

func hookDeletionAllowed(
	obj *unstructured.Unstructured,
) bool {
	switch obj.GetKind() {
	case "CustomResourceDefinition", "Namespace":
		return false

	default:
		return true
	}
}

func hooksForEvents(
	hooks []*release.Hook,
	events ...release.HookEvent,
) []*release.Hook {
	result := make(
		[]*release.Hook,
		0,
	)

	for _, hook := range hooks {
		for _, event := range events {
			if slices.Contains(
				hook.Events,
				event,
			) {
				result = append(
					result,
					hook,
				)

				break
			}
		}
	}

	return result
}

func HookObject(
	hook *release.Hook,
) (*unstructured.Unstructured, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(
		strings.NewReader(hook.Manifest),
		4096,
	)

	obj := &unstructured.Unstructured{}

	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf(
			"decode hook %q: %w",
			hook.Name,
			err,
		)
	}

	if len(obj.Object) == 0 {
		return nil, fmt.Errorf(
			"hook %q has an empty manifest",
			hook.Name,
		)
	}

	if obj.GetAPIVersion() == "" ||
		obj.GetKind() == "" ||
		obj.GetName() == "" {
		return nil, fmt.Errorf(
			"hook %q is missing apiVersion, kind, or metadata.name",
			hook.Name,
		)
	}

	return obj, nil
}

func validateHookAnnotations(
	annotations map[string]string,
) error {
	for _, raw := range strings.Split(
		annotations[release.HookAnnotation],
		",",
	) {
		event := strings.ToLower(
			strings.TrimSpace(raw),
		)

		if _, ok := supportedHookEvents[event]; !ok {
			return fmt.Errorf(
				"unsupported Helm bootstrap hook event %q",
				event,
			)
		}
	}

	if rawPolicies := annotations[release.HookDeleteAnnotation]; rawPolicies != "" {
		for _, raw := range strings.Split(
			rawPolicies,
			",",
		) {
			policy := strings.ToLower(
				strings.TrimSpace(raw),
			)

			if _, ok := supportedDeletePolicies[policy]; !ok {
				return fmt.Errorf(
					"unsupported Helm hook deletion policy %q",
					policy,
				)
			}
		}
	}

	return nil
}

func appendManifest(
	builder *strings.Builder,
	manifest string,
) {
	if builder.Len() > 0 {
		builder.WriteString("\n---\n")
	}

	builder.WriteString(
		strings.TrimSpace(manifest),
	)
	builder.WriteString("\n")
}
