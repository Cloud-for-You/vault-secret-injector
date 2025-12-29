# Nastavení HashiCorp Vault

## Nutná Hashicorp Vault ACL Policy

```
# list auth methods (for OIDC accessor discovery)
path "sys/auth"   { capabilities = ["read"] }
path "sys/auth/*" { capabilities = ["read"] }

# manage policies for per-NS access (create/update/list)
# manage policies (ACL API)
path "sys/policies/acl"   { capabilities = ["list","read"] }
path "sys/policies/acl/*" { capabilities = ["create","update","delete","list","read"] }

# enable/tune kv engines only (mount management)
path "sys/mounts"      { capabilities = ["list","read"] } 
path "sys/mounts/*"    { capabilities = ["create","update","delete","read","sudo"] }

# manage kubernetes auth roles
path "auth/k8s-kind/role/*" { capabilities = ["create","update","delete","list","read"] }

# manage identity groups & aliases (no secret reads)
path "identity/group"        { capabilities = ["create","update","list","read"] }
path "identity/group/name/*" { capabilities = ["read"] }
path "identity/group/id/*"   { capabilities = ["create","update","list","read"] }
path "identity/group-alias"  { capabilities = ["create","update","list","read"] }

# allow Vault CLI preflight (detect mount + kv version)
path "sys/internal/ui/mounts"   { capabilities = ["read"] }
path "sys/internal/ui/mounts/*" { capabilities = ["read"] }
```

## Vytvoření Hashicorp Vault Kubernetes role

```shell
kubectl port-forward -n hscp-vault svc/vault 8200:8200

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs. ......."
export VAULT_K8S_AUTH_MOUNT="k8s-kind"
export NAMESPACE="vault-secret-injector-system"
export SERVICEACCOUNT_NAME="vault-secret-injector-controller-manager"
export K8S_HOST="https://$(kubectl get svc -n default kubernetes -o jsonpath='{.spec.clusterIP}'):443"
export K8S_JWT=$(kubectl get secret -n vault-secret-injector-system vault-secret-injector-controller-manager-token -o jsonpath="{.data.token}" | base64 -d)
kubectl get secret -n vault-secret-injector-system vault-secret-injector-controller-manager-token -o jsonpath="{.data.ca\.crt}" | base64 -d > ca.crt

vault auth enable -path=${VAULT_K8S_AUTH_MOUNT} kubernetes
vault write auth/${VAULT_K8S_AUTH_MOUNT}/config token_reviewer_jwt=$K8S_JWT kubernetes_host=$K8S_HOST kubernetes_ca_cert=@ca.crt
vault write auth/${VAULT_K8S_AUTH_MOUNT}/role/vault-secret-injector \
  bound_service_account_names=${SERVICEACCOUNT_NAME} \
  bound_service_account_namespaces=${NAMESPACE} \
  policies=vault-secret-injector \
  audience=https://kubernetes.default.svc.cluster.local \
  alias_name_source=serviceaccount_uid