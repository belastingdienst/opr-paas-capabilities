# Paas-ArgoCD

## Inleiding
Wanneer binnen de `kind: Paas` ArgoCD wordt geconfigureerd wordt voor deze capability een namespace gecreeerd **abc-def-argocd** {{.metadata.name}}-argocd.

## Configuratie capability
Binnen de namespace **abc-def-argocd** wordt:
- Een instantie van ArgoCD uitgerold
- De applicationset controller is standaard ook beschikbaar, zodat je ook `ApplicationSet`s kan maken
- Een ArgoCD Application met de naam 'paas-bootstrap', geconfigureerd met de repo die in `.spec.capabilities.argocd.gitUrl` is opgegeven.
- De ArgoCD app wordt geconfigureerd met OpenShift RBAC, zodat je toegang wanneer je rechten op de Paas hebt.
- De controller gebruikt `resourceTrackingMethod: annotation+label`, waarmee resources een label en een annotatie krijgen van ArgoCD, zodat het andere software die vergelijkbare labels plaatst niet in de weg zit.