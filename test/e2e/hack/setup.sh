#!/usr/bin/env bash
# Prepares the current kube context (normally a fresh KinD cluster) for the capability e2e tests:
#
#   1. cert-manager (for the opr-paas webhooks)
#   2. CRDs: OpenShift mocks from opr-paas, plus the CRDs rendered by the capabilities
#   3. the opr-paas operator (released version) with the ArgoCD plugin generator enabled
#   4. upstream ArgoCD, configured to use the plugin generator
#   5. an in-cluster git server with a snapshot of the working tree
#   6. the capability ApplicationSets from this repository
#
# Environment:
#   CONTAINER_TOOL  docker (default) or podman
#   KIND            command to run kind (default: kind)
#   KIND_CLUSTER    name of the KinD cluster to load the git server image into
set -euo pipefail

CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
KIND="${KIND:-kind}"
KIND_CLUSTER="${KIND_CLUSTER:-paas-capabilities-e2e}"
CERTMANAGER_VERSION="${CERTMANAGER_VERSION:-v1.18.2}"

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(cd "${E2E_DIR}/../.." && pwd)"
GENERATED_DIR="${E2E_DIR}/.generated"
MANIFESTS="${E2E_DIR}/manifests"
GITSERVER_IMAGE="localhost/capabilities-gitserver:e2e"

step() { printf '\n==> %s\n' "$*"; }

# Only touch the disposable KinD e2e cluster, never whatever cluster happens to be the current context.
current_context="$(kubectl config current-context 2>/dev/null || true)"
if [[ "${E2E_ALLOW_ANY_CONTEXT:-false}" != "true" && "${current_context}" != "kind-${KIND_CLUSTER}" ]]; then
  echo "refusing to run: current context is '${current_context}', expected 'kind-${KIND_CLUSTER}'" >&2
  echo "(set E2E_ALLOW_ANY_CONTEXT=true to override)" >&2
  exit 1
fi

apply_ns() {
  kubectl create namespace "$1" --dry-run=client -o yaml | kubectl apply -f -
}

step "Installing cert-manager ${CERTMANAGER_VERSION}"
kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERTMANAGER_VERSION}/cert-manager.yaml"
for deploy in cert-manager cert-manager-cainjector cert-manager-webhook; do
  kubectl wait --for=condition=Available "deployment/${deploy}" -n cert-manager --timeout=180s
done
# The webhook can be Available before its TLS is ready; wait until a dry-run is accepted.
until echo '{"apiVersion":"cert-manager.io/v1","kind":"Issuer","metadata":{"name":"test"},"spec":{"selfSigned":{}}}' |
  kubectl create --dry-run=server -n cert-manager -f - >/dev/null 2>&1; do
  sleep 2
done

step "Installing CRDs"
kubectl kustomize "${MANIFESTS}/crds" | kubectl apply --server-side -f -
kubectl wait --for=condition=Established crd --all --timeout=60s

step "Creating secrets"
mkdir -p "${GENERATED_DIR}"
apply_ns paas-system
apply_ns argocd
token="$(openssl rand -hex 32)"
kubectl -n paas-system create secret generic generator-token \
  --from-literal=ARGOCD_GENERATOR_TOKEN="${token}" --dry-run=client -o yaml | kubectl apply -f -
kubectl -n argocd create secret generic paas-plugin-generator \
  --from-literal=token="${token}" --dry-run=client -o yaml |
  kubectl label --local -f - app.kubernetes.io/part-of=argocd -o yaml | kubectl apply -f -
# Decryption key for Paas secrets, referenced by the PaasConfig in main_test.go
openssl genrsa 2048 >"${GENERATED_DIR}/privateKey0" 2>/dev/null
kubectl -n paas-system create secret generic example-keys \
  --from-file=privateKey0="${GENERATED_DIR}/privateKey0" --dry-run=client -o yaml | kubectl apply -f -

step "Installing the opr-paas operator"
kubectl kustomize "${MANIFESTS}/opr-paas" | kubectl apply --server-side -f -
kubectl wait --for=condition=Available deployment/paas-controller-manager -n paas-system --timeout=180s

step "Installing ArgoCD"
kubectl kustomize "${MANIFESTS}/argocd" | kubectl apply --server-side -n argocd -f -
for deploy in argocd-repo-server argocd-applicationset-controller argocd-redis; do
  kubectl wait --for=condition=Available "deployment/${deploy}" -n argocd --timeout=300s
done
kubectl rollout status statefulset/argocd-application-controller -n argocd --timeout=300s

step "Building and loading the git server image"
build_dir="${GENERATED_DIR}/gitserver"
rm -rf "${build_dir}"
mkdir -p "${build_dir}/repos"
cp "${E2E_DIR}/gitserver/Dockerfile" "${E2E_DIR}/gitserver/main.go" "${build_dir}/"
# Snapshot of the working tree (tracked and new, non-ignored files), so local changes are tested too.
snapshot="$(mktemp -d)"
trap 'rm -rf "${snapshot}"' EXIT
(cd "${ROOT_DIR}" && git ls-files -co --exclude-standard -z | tar --null -T - -cf -) | tar -xf - -C "${snapshot}"
git -C "${snapshot}" init -q -b main
git -C "${snapshot}" add -A
git -C "${snapshot}" -c user.name=e2e -c user.email=e2e@example.invalid \
  commit -q --no-gpg-sign -m "e2e snapshot of the working tree"
git clone -q --bare "${snapshot}" "${build_dir}/repos/capabilities.git"
"${CONTAINER_TOOL}" build -t "${GITSERVER_IMAGE}" "${build_dir}"
"${CONTAINER_TOOL}" save "${GITSERVER_IMAGE}" -o "${GENERATED_DIR}/gitserver.tar"
${KIND} load image-archive "${GENERATED_DIR}/gitserver.tar" --name "${KIND_CLUSTER}"

step "Starting the git server"
kubectl apply -f "${MANIFESTS}/gitserver/gitserver.yaml"
# Restart so a re-run picks up a freshly loaded image with the same tag
kubectl -n e2e-git rollout restart deployment/git-server
kubectl -n e2e-git rollout status deployment/git-server --timeout=120s

step "Applying the capability ApplicationSets"
kubectl kustomize --load-restrictor LoadRestrictionsNone "${MANIFESTS}/capabilities" | kubectl apply -f -

step "Setup done; run the tests with: make e2e-test"
