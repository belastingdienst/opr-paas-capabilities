# Grafana as a capability

By adding Grafana as a capability all Paas requestors can also enable the `grafana` capability,
and get their very own Grafana to monitor all resources belonging to their Paas.

## Prerequisites

- Make sure K8s, ArgoCD and the Paas operator are [installed properly](../../README.md)
- Install the grafana operator:
  ```
  kubectl create -f https://github.com/grafana/grafana-operator/releases/latest/download/kustomize-cluster_scoped.yaml
  ```
- Install the extra resources as required by the Paas Capability integration
  ```
  kubectl apply -k ./deploy
  ```

## Adding Grafana as a capability

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
        grafana:
          custom_fields:
            image:
              validation: ^ghcr.io/myorg/something.*$
          default_permissions:
            grafana-paas-sa:
            - cluster-monitoring-view
          quotas:
            clusterwide: false
            defaults:
              limits.memory: 2Gi
              requests.cpu: "1"
              requests.memory: 1Gi
            ratio: 0
    ```

This would enable a capability called grafana, which means:
- that the plugin generator will respond to a request for Paas'es with the grafana capability enabled.
  The Paas can have `custom_fields` as defined in the `custom_fields` for the `grafana` capability in the PaasConfig.
  The plugin generator will have these values (either input in the Paas, or a templated value using go-templating as defined in the PaasConfig) as key/value pairs in the response. 
- Furthermore, for every paas with an enabled grafana capability, a namespace and cluster quota is created.

### Applicationset

For integration with ArgoCD, there needs to be an ApplicationSet with plugin generator generator configuration.
Use [this example](./applicationset.yaml) and change to your needs.
