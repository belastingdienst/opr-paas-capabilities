# Tekton as a capability

By adding Tekton as a capability all Paas requestors can also enable the Tekton capability,
and get their very own Tekton to manage all resources belonging to their Paas.
Main idea is to have all central Tekton Tasks available, as well as the option to enritch with your own Tekton Tasks and Pipelines.

## Prerequisites

- Make sure K8s, ArgoCD and the Paas operator are [installed properly](../../README.md)
- Install tekton pipelines:
  ```
  kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
  ```

## Adding Tekton as a capability

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
        tekton:
          default_permissions:
            pipeline:
            - monitoring-edit
            - alert-routing-edit
          quotas:
            clusterwide: true
            defaults:
              requests.cpu: "3"
              requests.memory: 5Gi
            min:
              requests.cpu: "2"
              requests.memory: 2Gi
            ratio: 0.20
    ```

This would enable a capability called tekton, which means:
- that the plugin generator will respond to a request for Paas'es with the tekton capability enabled.
  The Paas can have `custom_fields` as defined in the `custom_fields` for the `tekton` capability in the PaasConfig.
  The plugin generator will have these values (either input in the Paas, or a templated value using go-templating as defined in the PaasConfig) as key/value pairs in the response. 
- Furthermore, for every paas with an enabled tekton capability, a namespace and cluster quota is created.
  **note** that the ratio is set to 0.20 and clusterwide is set to true, which means that the tekton capability will use one cluster-wide quota and share resources.

### Applicationset

For integration with ArgoCD, there needs to be an ApplicationSet with plugin generator generator configuration.
Use [this example](./applicationset.yaml) and change to your needs.
