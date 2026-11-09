# End-to-end tests

These tests verify that the capabilities in this repository work with the
[Paas operator](https://github.com/belastingdienst/opr-paas) and ArgoCD, for the code you are working on.

They build on the [opr-paas e2e tests](https://github.com/belastingdienst/opr-paas/tree/main/test/e2e):
same framework ([`sigs.k8s.io/e2e-framework`](https://github.com/kubernetes-sigs/e2e-framework)),
the opr-paas API types, the OpenShift mocks, and a released opr-paas with the plugin generator enabled.
Unlike opr-paas, they also run a real ArgoCD, because that is what renders the capabilities.

## What is tested

For each capability:

1. A `Paas` with the capability enabled is created, using the `PaasConfig` from the capability README.
2. The opr-paas plugin generator feeds the capability `ApplicationSet` from this repository.
3. The central ArgoCD generates an `Application`, renders the capability from the code under test, and syncs it.
4. The tests assert on the rendered resources, and that everything is removed again with the `Paas`.

Capability operators (argocd-operator, grafana-operator, ...) are **not** installed; only their CRDs.
So the tests verify what a capability renders and that ArgoCD can apply it, not what the operators do with it.

Currently covered: `argocd`.

## How it works

```
KinD cluster
├── cert-manager                       webhooks of the Paas operator
├── CRDs                               OpenShift mocks (from opr-paas) + CRDs rendered by capabilities
├── paas-system   opr-paas v5.13.0     with the ArgoCD plugin generator on :4355
├── argocd        ArgoCD v3.1.12       AppProject + capability ApplicationSets from this repository
└── e2e-git       git server           snapshot of your working tree, served over HTTP
```

The capability ApplicationSets are applied as they are, except that their source is redirected to the
in-cluster git server (see [`manifests/capabilities`](./manifests/capabilities/kustomization.yaml)).
The git server contains a snapshot of the working tree (tracked and new, non-ignored files), so local
changes are tested without committing or pushing them.

## Running locally

Requirements: Go, kubectl, docker or podman.

```
make e2e             # create the cluster, set it up and run the tests
make e2e-test        # run the tests again
make e2e-setup       # after changing a capability: refresh the git snapshot (and re-apply the rest)
make e2e-clean       # delete the cluster
```

Use `CONTAINER_TOOL=podman` with podman.

The make targets use their own kubeconfig (`test/e2e/.generated/kubeconfig`) and never touch the
clusters in your `~/.kube/config`. As an extra safeguard, the setup script and the tests refuse to run
unless the current context is `kind-paas-capabilities-e2e` (override with `E2E_ALLOW_ANY_CONTEXT=true`).

## Adding a capability

1. Add the CRDs the capability renders to [`manifests/crds`](./manifests/crds/kustomization.yaml).
2. Add its `ApplicationSet` and a source patch to [`manifests/capabilities`](./manifests/capabilities/kustomization.yaml).
3. Add its `PaasConfig` capability, as documented in its README, to [`main_test.go`](./main_test.go).
4. Add a `capability_<name>_test.go`.
