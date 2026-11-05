# Central ArgoCD: quickstart install

The manifests in this directory install the central ArgoCD that rolls out the capabilities.
They are meant for a **local test cluster** (see the [quickstart](../../README.md#quickstart)) and to
take inspiration from. They are **not** a secure production setup.

## What you must change before using this outside a test cluster

| File | Quickstart setting | Why it is risky | What to do instead |
|------|--------------------|-----------------|--------------------|
| [`aproj_paas.yaml`](./aproj_paas.yaml) | The `paas` AppProject allows all resource kinds (`'*'`), all destinations and all source repositories. | Every capability is deployed through this project. If a capability chart can be tricked into rendering an extra resource (for example through an unvalidated custom field), ArgoCD applies it anywhere in the cluster. | Allow only the source repositories of your capability charts, only the namespaces capabilities deploy to (for example `*-argocd`, `*-grafana`), and list the cluster-scoped kinds you actually need in `clusterResourceWhitelist`. |
| [`crb_cluster-admin-argocd-argocd-application-controller.yaml`](./crb_cluster-admin-argocd-argocd-application-controller.yaml) | The application-controller of the central ArgoCD is cluster-admin. | Whatever ends up in a capability Application is applied with full cluster rights. | Bind a ClusterRole that only contains the resources and verbs your capabilities need. |
| [`s_argo-service-cluster.yaml`](./s_argo-service-cluster.yaml) | The ArgoCD admin password is a fixed, well-known value. | Anyone can log in as ArgoCD admin. | Let ArgoCD generate the initial admin password, or provide it from your own secret management. Disable the local admin user when you use SSO. |
| [`g_clusteradmin.yaml`](./g_clusteradmin.yaml) | The `kubeadmin` and `developer` users are in the `clusteradmin` group. | Grants cluster-admin to test users. | Use your own groups and identity provider. |

These measures limit the damage. They do not replace validating input: also follow the advice for
custom fields in each capability, for example [Validate custom fields](../../capabilities/argocd/README.md#validate-custom-fields)
for the ArgoCD capability.
