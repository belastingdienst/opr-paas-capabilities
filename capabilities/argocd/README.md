# ArgoCD as a capability

By adding ArgoCD as a capability all Paas requestors can also enable the ArgoCD capability,
and get their very own ArgoCD to manage all resources belonging to their Paas.

Additionally, a Paas can be configured to be managed by the ArgoCD of another Paas.

## Prerequisites

- Make sure K8s, ArgoCD and the Paas operator are [installed properly](../../README.md)

## Adding ArgoCD as a capability

### Paas Config changes

Add the following to your Paas config block:

!!! example
    ```yaml
    apiVersion: cpet.belastingdienst.nl/v1alpha2
    kind: PaasConfig
    metadata:
      name: paas-config
    ...
    spec:
      capabilities:
        argocd:
          custom_fields:
            git_path:
              default: .
              validation: '^[A-Za-z0-9._/-]+$'
            git_revision:
              validation: '^[A-Za-z0-9._/-]+$'
            git_url:
              validation: '^(https://git\.example\.com/|ssh://git@git\.example\.com/)[A-Za-z0-9._/-]+$'
            groups:
              default: ""
              required: false
              template: |
                g, {{ .Paas.Name }}-admins, role:admin
              validation: ""
          default_permissions:
            argocd-argocd-application-controller:
            - monitoring-edit
            - alert-routing-edit
          quotas:
            clusterwide: true
            defaults:
              requests.cpu: "1"
              requests.memory: 5Gi
              requests.storage: 1Gi
            ratio: 0.01
    ```

This would enable a capability called argocd, which means:
- that the plugin generator will respond to a request for Paas'es with the argocd capability enabled.
  The Paas can have `custom_fields` as defined in the `custom_fields` for the `argocd` capability in the PaasConfig.
  The plugin generator will have these values (either input in the Paas, or a templated value using go-templating as defined in the PaasConfig) as key/value pairs in the response. 
- Furthermore, for every paas with an enabled argocd capability, a namespace and cluster quota is created.

### Validate custom fields

Custom fields are input from Paas owners. Their values end up in the Helm chart that the central
ArgoCD renders and applies with its own (broad) permissions. Always restrict them as tightly as you can:

- Give every custom field a `validation` regex that allows only what you expect. The example above:
  - `git_url`: only repositories on your own git server (replace `git.example.com`), over https or ssh.
  - `git_path` and `git_revision`: a single line of letters, digits, `.`, `_`, `/` and `-`.
- Anchor the regex with `^` and `$`, so the whole value is checked and not only a part of it.
- Never allow newlines or other control characters. A multi-line value could otherwise add extra YAML
  documents, and with them arbitrary resources, to what ArgoCD applies.
- Prefer a `template` over free input for values Paas owners should not choose themselves, as is done for `groups`.

The chart has a second line of defence: it quotes all values and fails to render when a single-line
value contains control characters. [`chart/tests/injection_test.sh`](./chart/tests/injection_test.sh)
checks this. Keep both defences when you adapt this capability.

### Applicationset

For integration with ArgoCD, there needs to be an ApplicationSet with plugin generator generator configuration.
Use [this example](./applicationset.yaml) and change to your needs.
