/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// Package e2e verifies the capabilities in this repository end to end: a Paas is created, the
// opr-paas plugin generator feeds the capability ApplicationSet, and the central ArgoCD renders
// and syncs the capability from the code under test (served by the in-cluster git server).
//
// The cluster is prepared by hack/setup.sh; see test/e2e/README.md.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	// gitServerURL is where hack/setup.sh serves the snapshot of the working tree.
	gitServerURL = "http://git-server.e2e-git.svc.cluster.local/capabilities.git"
	// argocdNamespace is where the central ArgoCD and the capability ApplicationSets live.
	argocdNamespace = "argocd"
)

var (
	testenv env.Environment

	// paasConfig mirrors the PaasConfig examples in the capability READMEs. When you change a README
	// example, change this config too, so the documented configuration is what gets tested.
	paasConfig = v1alpha2.PaasConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "paas-config"},
		Spec: v1alpha2.PaasConfigSpec{
			DecryptKeysSecret: v1alpha2.NamespacedName{Name: "example-keys", Namespace: "paas-system"},
			// The CRD defaults for these fields do not apply when creating from Go: the fields have no
			// omitempty, so "" is sent and the namespaces get a label with an empty key. Set the defaults.
			QuotaLabel:      "clusterquotagroup",
			RequestorLabel:  "requestor",
			ManagedByLabel:  "argocd.argoproj.io/managed-by",
			ManagedBySuffix: "argocd",
			Capabilities: v1alpha2.ConfigCapabilities{
				// capabilities/argocd/README.md
				"argocd": {
					CustomFields: map[string]v1alpha2.ConfigCustomField{
						"git_url": {
							// The README allows only your own git server; here that is the e2e git server.
							Validation: `^(http://git-server\.e2e-git\.svc\.cluster\.local/)[A-Za-z0-9._/-]+$`,
						},
						"git_path": {
							Default:    ".",
							Validation: `^[A-Za-z0-9._/-]+$`,
						},
						"git_revision": {
							Validation: `^[A-Za-z0-9._/-]+$`,
						},
						"groups": {
							Template: "g, {{ .Paas.Name }}-admins, role:admin\n",
						},
					},
					QuotaSettings: v1alpha2.ConfigQuotaSettings{
						Clusterwide: true,
						DefQuota: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceRequestsCPU:     resource.MustParse("1"),
							corev1.ResourceRequestsMemory:  resource.MustParse("5Gi"),
							corev1.ResourceRequestsStorage: resource.MustParse("1Gi"),
						},
						Ratio: 0.01,
					},
				},
			},
			Templating: v1alpha2.ConfigTemplatingItems{
				// Used by the capability ApplicationSets (requestor, service, subservice).
				GenericCapabilityFields: v1alpha2.ConfigTemplatingItem{
					"requestor":  "{{ .Paas.Spec.Requestor }}",
					"service":    `{{ (split "-" .Paas.Name)._0 }}`,
					"subservice": `{{ (split "-" .Paas.Name)._1 }}`,
				},
			},
		},
	}
)

func TestMain(m *testing.M) {
	kubeconfig := conf.ResolveKubeConfigFile()
	if err := checkContext(kubeconfig); err != nil {
		fmt.Fprintf(os.Stderr, "refusing to run e2e tests: %v\n", err)
		os.Exit(1)
	}

	cfg := envconf.NewWithKubeConfig(kubeconfig)
	testenv = env.NewWithConfig(cfg)

	if err := registerSchemes(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to register schemes: %v\n", err)
		os.Exit(1)
	}

	testenv.Setup(func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Not paasConfig.DeepCopy(): quota.Quota.DeepCopy() turns nil min/max quotas into empty maps,
		// which the PaasConfig webhook rejects (opr-paas v5.13.0).
		config := &paasConfig
		if err := cfg.Client().Resources().Create(ctx, config); err != nil {
			return ctx, fmt.Errorf("failed to create PaasConfig: %w", err)
		}
		if err := waitForPaasConfigActive(ctx, cfg, config); err != nil {
			return ctx, err
		}
		// Leftovers of an interrupted run; they can only be removed while a PaasConfig exists.
		return ctx, deleteE2EPaases(ctx, cfg)
	})

	testenv.Finish(func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		// Paases first: the operator needs the PaasConfig to finalize them.
		if err := deleteE2EPaases(ctx, cfg); err != nil {
			return ctx, err
		}
		return ctx, deleteResourceSync(ctx, cfg, &v1alpha2.PaasConfig{ObjectMeta: metav1.ObjectMeta{Name: "paas-config"}})
	})

	os.Exit(testenv.Run(m))
}

// checkContext makes sure the tests only run against a disposable cluster: they create and delete
// cluster-scoped resources such as the PaasConfig. The current context must be the KinD e2e cluster,
// unless E2E_ALLOW_ANY_CONTEXT=true is set.
func checkContext(kubeconfig string) error {
	if os.Getenv("E2E_ALLOW_ANY_CONTEXT") == "true" {
		return nil
	}
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	raw, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).RawConfig()
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig %q: %w", kubeconfig, err)
	}
	expected := "kind-" + getenv("KIND_CLUSTER", "paas-capabilities-e2e")
	if raw.CurrentContext != expected {
		return fmt.Errorf("current context is %q, expected %q (set E2E_ALLOW_ANY_CONTEXT=true to override)",
			raw.CurrentContext, expected)
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
