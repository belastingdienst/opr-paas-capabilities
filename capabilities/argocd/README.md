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
            git_revision: {}
            git_url: {}
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

### Applicationset

For integration with ArgoCD, there needs to be an ApplicationSet with plugin generator generator configuration.
Use [this example](./applicationset.yaml) and change to your needs.
