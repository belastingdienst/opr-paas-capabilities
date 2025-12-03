# opr-paas-capabilities

## Goal

The [PaaS operator](https://github.com/belastingdienst/opr-paas) is meant to deliver all that one application requires,
such as:

- one or more static application namespaces
- one or more dynamic application namespaces (for e2e tests of pull requests on the app including all requirements of the app)
- capabilities such as tekton, ArgoCD, grafana, etc.
- security (group permissions, Cluster Role Bindings, etc.)
- quota

Paas Capabilities are freely programmable, and this repo is used as a catalogue for community members
that would like to develop and contribute capabilities.

## Quickstart

- Create a k8s and get access to it:
  Kind:

  - Install [KinD](https://kind.sigs.k8s.io/docs/user/quick-start)
    ```
    go install sigs.k8s.io/kind@v0.30.0
    kind create cluster
    ```
  - `kind create cluster --name paas`

- Install ArgoCD
  `kubectl apply -k install/argocd`

- Install the Paas operator:
  ```
  kubectl apply -f https://github.com/belastingdienst/opr-paas/releases/latest/download/install.yaml
  kubectl apply -f https://raw.githubusercontent.com/belastingdienst/opr-paas/refs/heads/main/examples/resources/_v1alpha2_paasconfig.yaml
  ```

## Contributing

Please refer to our documentation in the [CONTRIBUTING.md](./CONTRIBUTING.md) file
and the Developer Guide section of the documentation site if you want to help us
improve the Paas Operator.

## License

Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.

See [LICENSE.md](./LICENSE.md) for details.
