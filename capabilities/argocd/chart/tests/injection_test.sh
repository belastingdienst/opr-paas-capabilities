#!/usr/bin/env bash
# Regression tests: values that come from Paas custom fields must never be able to break out of
# their YAML field (CVD fix). Run from anywhere: ./tests/injection_test.sh
set -euo pipefail

CHART_DIR="$(cd "$(dirname "$0")/.." && pwd)"
failures=0

render() {
  helm template test "${CHART_DIR}" --set-string git_url=https://example.com/repo.git "$@"
}

pass() { echo "PASS: $1"; }
fail() { echo "FAIL: $1"; failures=$((failures + 1)); }

injected=$'main\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding\nmetadata:\n  name: injected'

# 1. A multi-line value in a single-line field must make the render fail.
for field in git_url git_path git_revision oplosgroep; do
  if render --set-string "${field}=${injected}" >/dev/null 2>&1; then
    fail "${field}: multi-line value was accepted"
  else
    pass "${field}: multi-line value is rejected"
  fi
done

# 2. YAML special characters on a single line must be kept as a literal string.
out="$(render --set-string 'git_revision=main # comment: {x: y}')"
if grep -qF 'targetRevision: "main # comment: {x: y}"' <<<"${out}"; then
  pass "git_revision: special characters are quoted"
else
  fail "git_revision: special characters are not quoted"
fi

# 3. groups is multi-line by design; it must stay inside the policy field.
groups_file="$(mktemp)"
trap 'rm -f "${groups_file}"' EXIT
printf 'g, admins, role:admin\n%s\n' "${injected}" >"${groups_file}"
out="$(render --set-file "groups=${groups_file}")"
if grep -q '^kind: ClusterRoleBinding' <<<"${out}" || grep -q 'name: injected$' <<<"${out}"; then
  fail "groups: injected document was rendered"
else
  pass "groups: value stays inside the policy field"
fi

if [[ ${failures} -gt 0 ]]; then
  echo "${failures} test(s) failed"
  exit 1
fi
echo "all tests passed"
