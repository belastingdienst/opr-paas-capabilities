/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/pkg/quota"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	argoPaas        = "e2e-argocd"
	argoNamespace   = argoPaas + "-argocd"
	argoApplication = argoPaas + "-capability-argocd" // capabilities/argocd/applicationset.yaml
	argoGitURL      = gitServerURL
	argoGitPath     = "capabilities/argocd/chart"
	argoGitRevision = "main"
	argoRequestor   = "e2e-requestor"
)

func argoPaasSpec(customFields map[string]string) v1alpha2.PaasSpec {
	return v1alpha2.PaasSpec{
		Requestor: argoRequestor,
		Quota:     quota.Quota{corev1.ResourceLimitsCPU: resource.MustParse("1")},
		Capabilities: v1alpha2.PaasCapabilities{
			"argocd": v1alpha2.PaasCapability{CustomFields: customFields},
		},
	}
}

// TestCapabilityArgoCD verifies the chain Paas -> plugin generator -> ApplicationSet -> Application
// -> rendered chart for the argocd capability.
func TestCapabilityArgoCD(t *testing.T) {
	spec := argoPaasSpec(map[string]string{
		"git_url":      argoGitURL,
		"git_path":     argoGitPath,
		"git_revision": argoGitRevision,
	})

	testenv.Test(
		t,
		features.New("ArgoCD capability").
			Setup(createPaasFn(argoPaas, spec)).
			Assess("Application is generated and synced", assertArgoApplicationSynced).
			Assess("ArgoCD instance is rendered in the capability namespace", assertArgoInstance).
			Assess("paas-bootstrap Application uses the custom fields", assertArgoBootstrap).
			Assess("Application is removed with the Paas", assertArgoRemoved).
			Teardown(teardownPaasFn(argoPaas)).
			Feature(),
	)
}

func assertArgoApplicationSynced(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	app := waitForApplicationSynced(ctx, t, cfg, argoApplication)

	assert.Equal(t, argoNamespace, mustNestedString(t, app, "spec", "destination", "namespace"))
	assert.Equal(t, "paas", mustNestedString(t, app, "spec", "project"))
	return ctx
}

func assertArgoInstance(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	argocd := waitForObject(ctx, t, cfg, argoCDGVK, argoNamespace, "argo-service")

	// groups comes from the PaasConfig template and ends up in the RBAC policy.
	policy := mustNestedString(t, argocd, "spec", "rbac", "policy")
	assert.Contains(t, policy, "g, "+argoPaas+"-admins, role:admin", "RBAC policy of %s", describe(argocd))
	return ctx
}

func assertArgoBootstrap(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	bootstrap := waitForObject(ctx, t, cfg, applicationGVK, argoNamespace, "paas-bootstrap")

	assert.Equal(t, argoGitURL, mustNestedString(t, bootstrap, "spec", "source", "repoURL"))
	assert.Equal(t, argoGitPath, mustNestedString(t, bootstrap, "spec", "source", "path"))
	assert.Equal(t, argoGitRevision, mustNestedString(t, bootstrap, "spec", "source", "targetRevision"))
	assert.Equal(t, argoRequestor, bootstrap.GetLabels()["itsmoplosgroep"])
	return ctx
}

func assertArgoRemoved(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	require.NoError(t, deleteResourceSync(ctx, cfg, &v1alpha2.Paas{ObjectMeta: metav1.ObjectMeta{Name: argoPaas}}))
	require.NoError(t, waitForGone(ctx, cfg, applicationGVK, argocdNamespace, argoApplication),
		"the ApplicationSet should remove the Application once the Paas is gone")
	return ctx
}

// TestCapabilityArgoCDRejectsMultiLineCustomFields verifies that the validation regexes documented in
// capabilities/argocd/README.md stop a custom field value from injecting extra YAML documents.
func TestCapabilityArgoCDRejectsMultiLineCustomFields(t *testing.T) {
	injected := "main\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: injected"

	testenv.Test(
		t,
		features.New("ArgoCD capability input validation").
			Assess("multi-line custom fields are rejected", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
				for _, field := range []string{"git_url", "git_path", "git_revision"} {
					fields := map[string]string{
						"git_url":      argoGitURL,
						"git_path":     argoGitPath,
						"git_revision": argoGitRevision,
					}
					fields[field] = fields[field] + injected
					paas := &v1alpha2.Paas{
						ObjectMeta: metav1.ObjectMeta{Name: "e2e-reject-" + strings.ReplaceAll(field, "_", "")},
						Spec:       argoPaasSpec(fields),
					}
					err := cfg.Client().Resources().Create(ctx, paas)
					if !assert.Error(t, err, "Paas with a multi-line %s must be rejected", field) {
						_ = deleteResourceSync(ctx, cfg, paas)
						continue
					}
					assert.Contains(t, err.Error(), field, "the rejection should name the field")
				}
				return ctx
			}).
			Feature(),
	)
}
