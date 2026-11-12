# End-to-end tests for the capabilities in this repository. See test/e2e/README.md.

KIND_CLUSTER ?= paas-capabilities-e2e
KIND_VERSION ?= v0.30.0
KIND ?= go run sigs.k8s.io/kind@$(KIND_VERSION)
CONTAINER_TOOL ?= docker

# The e2e targets use their own kubeconfig, so they never touch the clusters in ~/.kube/config.
E2E_KUBECONFIG ?= $(CURDIR)/test/e2e/.generated/kubeconfig
E2E_ENV = KUBECONFIG=$(E2E_KUBECONFIG) KIND="$(KIND)" KIND_CLUSTER=$(KIND_CLUSTER) CONTAINER_TOOL=$(CONTAINER_TOOL)

ifeq ($(CONTAINER_TOOL),podman)
KIND_PROVIDER = KIND_EXPERIMENTAL_PROVIDER=podman
endif

.PHONY: e2e
e2e: e2e-cluster e2e-setup e2e-test ## Create a KinD cluster, set it up and run the e2e tests

.PHONY: e2e-cluster
e2e-cluster: ## Create the KinD cluster for the e2e tests
	mkdir -p $(dir $(E2E_KUBECONFIG))
	$(KIND_PROVIDER) $(KIND) create cluster --name $(KIND_CLUSTER) --kubeconfig $(E2E_KUBECONFIG)

.PHONY: e2e-setup
e2e-setup: ## Install cert-manager, opr-paas, ArgoCD, the git server and the capabilities
	$(E2E_ENV) $(KIND_PROVIDER) test/e2e/hack/setup.sh

.PHONY: e2e-test
e2e-test: ## Run the e2e tests against the KinD cluster
	cd test/e2e && $(E2E_ENV) go test -count=1 -v -timeout 30m .

.PHONY: e2e-clean
e2e-clean: ## Delete the KinD cluster and generated files
	$(KIND_PROVIDER) $(KIND) delete cluster --name $(KIND_CLUSTER) --kubeconfig $(E2E_KUBECONFIG) || true
	rm -rf test/e2e/.generated

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'
