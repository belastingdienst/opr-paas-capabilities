/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package e2e

// Helpers adapted from the opr-paas e2e tests (test/e2e/util.go and steps.go in
// github.com/belastingdienst/opr-paas). They are unexported there, so they are copied here.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	quotav1 "github.com/openshift/api/quota/v1"
	userv1 "github.com/openshift/api/user/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

const (
	waitInterval = 2 * time.Second
	// Paas reconciliation is fast; ArgoCD needs time to generate, render and sync.
	paasTimeout   = 1 * time.Minute
	argocdTimeout = 5 * time.Minute
	// Removing an Application can take several minutes: once the operator has deleted the capability
	// namespace, the Application controller backs off its reconciles (10s, 20s, ... 160s) and only then
	// processes the deletion.
	removalTimeout = 10 * time.Minute
)

var (
	applicationGVK = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"}
	argoCDGVK      = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1beta1", Kind: "ArgoCD"}
)

func registerSchemes(cfg *envconf.Config) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}
	scheme := r.GetScheme()
	for _, install := range []func(*runtime.Scheme) error{
		v1alpha2.AddToScheme,
		quotav1.Install,
		userv1.Install,
	} {
		if err := install(scheme); err != nil {
			return err
		}
	}
	return nil
}

func waitFor(ctx context.Context, condition apimachinerywait.ConditionWithContextFunc, timeout time.Duration) error {
	return wait.For(condition, wait.WithContext(ctx), wait.WithTimeout(timeout), wait.WithInterval(waitInterval))
}

// waitForPaasConfigActive blocks until the operator has accepted the PaasConfig as the active one.
func waitForPaasConfigActive(ctx context.Context, cfg *envconf.Config, config *v1alpha2.PaasConfig) error {
	return waitFor(ctx, conditions.New(cfg.Client().Resources()).ResourceMatch(config, func(obj k8s.Object) bool {
		return meta.IsStatusConditionTrue(obj.(*v1alpha2.PaasConfig).Status.Conditions, v1alpha2.TypeActivePaasConfig)
	}), paasTimeout)
}

// deleteResourceSync deletes the resource and returns once it is gone.
func deleteResourceSync(ctx context.Context, cfg *envconf.Config, obj k8s.Object) error {
	if err := cfg.Client().Resources().Delete(ctx, obj); err != nil {
		return err
	}
	return waitFor(ctx, conditions.New(cfg.Client().Resources()).ResourceDeleted(obj), argocdTimeout)
}

// deleteE2EPaases deletes all Paases created by these tests (name prefix "e2e-") and waits until they are gone.
func deleteE2EPaases(ctx context.Context, cfg *envconf.Config) error {
	var list v1alpha2.PaasList
	if err := cfg.Client().Resources().List(ctx, &list); err != nil {
		return fmt.Errorf("failed to list Paases: %w", err)
	}
	for i := range list.Items {
		paas := &list.Items[i]
		if !strings.HasPrefix(paas.Name, "e2e-") {
			continue
		}
		if err := deleteResourceSync(ctx, cfg, paas); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Paas %s: %w", paas.Name, err)
		}
	}
	return nil
}

// createPaasFn creates a Paas and waits until the operator reports it Ready.
func createPaasFn(name string, spec v1alpha2.PaasSpec) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		paas := &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: spec}
		if err := cfg.Client().Resources().Create(ctx, paas); err != nil {
			t.Fatalf("failed to create Paas %s: %v", name, err)
		}
		err := waitFor(ctx, conditions.New(cfg.Client().Resources()).ResourceMatch(paas, func(obj k8s.Object) bool {
			p := obj.(*v1alpha2.Paas)
			ready := meta.FindStatusCondition(p.Status.Conditions, v1alpha2.TypeReadyPaas)
			return ready != nil && ready.Status == metav1.ConditionTrue && ready.ObservedGeneration == p.Generation
		}), paasTimeout)
		if err != nil {
			t.Fatalf("Paas %s did not become Ready: %v (conditions: %+v)", name, err, paas.Status.Conditions)
		}
		return ctx
	}
}

// teardownPaasFn deletes the Paas if it still exists.
func teardownPaasFn(name string) types.StepFunc {
	return func(ctx context.Context, _ *testing.T, cfg *envconf.Config) context.Context {
		_ = deleteResourceSync(ctx, cfg, &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: name}})
		return ctx
	}
}

func newUnstructured(gvk schema.GroupVersionKind, namespace, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetNamespace(namespace)
	obj.SetName(name)
	return obj
}

// waitForObject waits until the object exists and returns it.
func waitForObject(
	ctx context.Context, t *testing.T, cfg *envconf.Config, gvk schema.GroupVersionKind, namespace, name string,
) *unstructured.Unstructured {
	t.Helper()
	obj := newUnstructured(gvk, namespace, name)
	err := waitFor(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Client().Resources().Get(ctx, name, namespace, obj); err != nil {
			return false, nil //nolint:nilerr // not found yet, keep polling
		}
		return true, nil
	}, argocdTimeout)
	if err != nil {
		t.Fatalf("%s %s/%s was not created: %v", gvk.Kind, namespace, name, err)
	}
	return obj
}

// waitForApplicationSynced waits until an ArgoCD Application reports sync status Synced.
// On timeout the Application status is included in the error, which usually explains why.
func waitForApplicationSynced(ctx context.Context, t *testing.T, cfg *envconf.Config, name string) *unstructured.Unstructured {
	t.Helper()
	app := newUnstructured(applicationGVK, argocdNamespace, name)
	err := waitFor(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Client().Resources().Get(ctx, name, argocdNamespace, app); err != nil {
			return false, nil //nolint:nilerr // not found yet, keep polling
		}
		status, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
		return status == "Synced", nil
	}, argocdTimeout)
	if err != nil {
		status, _, _ := unstructured.NestedMap(app.Object, "status")
		t.Fatalf("Application %s is not Synced: %v\nstatus: %v", name, err, status)
	}
	return app
}

// waitForGone waits until the object no longer exists.
func waitForGone(ctx context.Context, cfg *envconf.Config, gvk schema.GroupVersionKind, namespace, name string) error {
	return waitFor(ctx, conditions.New(cfg.Client().Resources()).
		ResourceDeleted(newUnstructured(gvk, namespace, name)), removalTimeout)
}

func mustNestedString(t *testing.T, obj *unstructured.Unstructured, fields ...string) string {
	t.Helper()
	value, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil || !found {
		t.Fatalf("field %v not found in %s %s: %v", fields, obj.GetKind(), obj.GetName(), err)
	}
	return value
}

func describe(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s %s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
}
