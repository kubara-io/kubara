package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	kubarahelm "github.com/kubara-io/kubara/internal/helm"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/rs/zerolog/log"
	release "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

type HookOptions struct {
	Timeout      time.Duration
	ApplyOptions k8s.ApplyOptions
}

type hookExecutor struct {
	client hookClient
	opts   HookOptions
}

func NewHooks(
	client hookClient,
	opts HookOptions,
) Hooks {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}

	return &hookExecutor{
		client: client,
		opts:   opts,
	}
}

func (h *hookExecutor) PreApply(
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

func (h *hookExecutor) PostApply(
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

func (h *hookExecutor) PreDelete(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPreDelete,
	)
}

func (h *hookExecutor) PostDelete(
	ctx context.Context,
	hooks []*release.Hook,
) error {
	return h.run(
		ctx,
		hooks,
		release.HookPostDelete,
	)
}

func (h *hookExecutor) run(
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

		obj, err := kubarahelm.HookObject(hook)
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

func (h *hookExecutor) failHook(
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

func (h *hookExecutor) DeleteByPolicy(
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

	obj, err := kubarahelm.HookObject(hook)
	if err != nil {
		return err
	}

	if !bootstrapHookDeletionAllowed(obj) {
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
	// Helm defaults a missing delete policy to before-hook-creation.
	if len(hook.DeletePolicies) == 0 {
		return policy == release.HookBeforeHookCreation
	}

	return slices.Contains(
		hook.DeletePolicies,
		policy,
	)
}

func bootstrapHookDeletionAllowed(
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

				// A hook declaring pre-install and pre-upgrade belongs
				// to the same Kubara pre-apply phase and runs once.
				break
			}
		}
	}

	return result
}
