# vault-secret-injector

## Description
Operátor, který poskytuje komplexní řešení pro správu tajných údajů v Kubernetes prostředí pomocí HashiCorp Vault. Projekt nabízí:

- **Automatickou synchronizaci tajných údajů**: Operator kontinuálně synchronizuje data z Vault do Kubernetes Secrets na základě definovaných VaultSecret CRD
- **Bezpečnou správu citlivých dat**: Centralizovaná správa tajných údajů s pokročilými bezpečnostními funkcemi Vault
- **Izolaci na úrovni Namespace**: Každý Namespace má své vlastní Vault politiky a autentizační role
- **Validaci a automatizaci**: Integrované validační webhooky zajišťují správnou konfiguraci a automatickou správu Vault zdrojů

### Výhody a použití
Tento operator je ideální pro organizace, které potřebují:
- Centralizovanou správu tajných údajů napříč více aplikacemi a týmy, ale nechtějí implementovat plný Vault Secret Operátor z důvodu složité konfigurace
- Automatickou rotaci a aktualizaci credentials bez výpadků služeb
- Dodržování bezpečnostních standardů s auditováním přístupu k tajným údajům
- Snadnou integraci s existujícími Kubernetes deploymenty a CI/CD pipelines
- Izolaci tajných údajů mezi různými prostředími (dev, staging, prod)

### Validating Webhook
Operator obsahuje validační webhook pro Namespace zdroje, který automaticky spravuje Vault zdroje při životním cyklu Namespace. Tato funkce přináší uživatelům následující výhody:

- **Automatická izolace tajných údajů**: Každý Namespace získá své vlastní Vault politiky a role bez manuální konfigurace
- **Zabezpečení podle principu nejmenších privilegií**: Service Accounts v Namespace mohou přistupovat pouze k tajným údajům svého Namespace
- **Automatické čištění**: Při smazání Namespace se automaticky odstraní související Vault zdroje, čímž se předchází zanechání osiřelých politik
- **Auditovatelnost**: Všechny akce webhooku jsou auditovány ve Vault pro sledování změn
- **Zjednodušení správy**: Eliminuje potřebu manuální správy Vault politik pro každý Namespace

Webhook provádí následující akce při vytváření, aktualizaci nebo mazání Namespace:

- **Při vytváření Namespace**: Webhook se autentizuje do HashiCorp Vault pomocí Kubernetes autentizace, vytvoří Vault politiku umožňující čtení a výpis tajných údajů v cestě `secret/data/{namespace}/*`, a vytvoří Kubernetes autentizační roli ve Vault vázanou na daný Namespace s touto politikou.
- **Při aktualizaci Namespace**: V současné době neprovádí žádné akce (vyhrazeno pro budoucí rozšíření).
- **Při mazání Namespace**: Webhook se autentizuje do Vault, smaže odpovídající Kubernetes autentizační roli a politiku spojenou s Namespace.

### Hashicorp Vault ACL Policy
```
# list auth methods (for OIDC accessor discovery)
path "sys/auth"   { capabilities = ["read"] }
path "sys/auth/*" { capabilities = ["read"] }

# manage policies for per-NS access (create/update/list)
# manage policies (ACL API)
path "sys/policies/acl"   { capabilities = ["list","read"] }
path "sys/policies/acl/*" { capabilities = ["create","update","delete","list","read"] }
path "sys/policy/*"       { capabilities = ["create","update","delete","list","read"] }

# enable/tune kv engines only (mount management)
path "sys/mounts"      { capabilities = ["list","read"] } 
path "sys/mounts/*"    { capabilities = ["create","update","delete","read","sudo"] }
path "sys/mounts/kv-*" { capabilities = ["create","update","delete","read","sudo"] }

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

### Hashicorp Vault Kubernetes role
```shell
kubectl port-forward -n hscp-vault svc/vault 8200:8200

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs. ......."
export CLUSTER_NAME="k8s-kind"
export K8S_HOST="https://$(kubectl get svc -n default kubernetes -o jsonpath='{.spec.clusterIP}'):443"
export K8S_JWT=$(kubectl get secret -n vault-secret-injector-system vault-secret-injector-controller-manager-token -o jsonpath="{.data.token}" | base64 -d)
kubectl get secret -n vault-secret-injector-system vault-secret-injector-controller-manager-token -o jsonpath="{.data.ca\.crt}" | base64 -d > ca.crt

vault auth enable -path=${CLUSTER_NAME} kubernetes
vault write auth/${CLUSTER_NAME}/config token_reviewer_jwt=$K8S_JWT kubernetes_host=$K8S_HOST kubernetes_ca_cert=@ca.crt
vault write auth/${CLUSTER_NAME}/role/vault-secret-injector \
  bound_service_account_names=vault-secret-injector-controller-manager \
  bound_service_account_namespaces=vault-secret-injector-system \
  policies=vault-secret-injector \
  audience=https://kubernetes.default.svc.cluster.local \
  alias_name_source=serviceaccount_uid
```


### Reconciliation Custom CRD
Operator implementuje reconciliation smyčku pro VaultSecret CRD, která zajišťuje kontinuální synchronizaci tajných údajů z Vault do Kubernetes Secrets. Příkladem použití je VaultSecret CRD:

```yaml
apiVersion: cfy.cz/v1
kind: VaultSecret
metadata:
  name: vaultsecret-sample
spec:
  stringData:
    secret-key: secret/data/myapp/config#password
```

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/vault-secret-injector:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/vault-secret-injector:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/vault-secret-injector:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/vault-secret-injector/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
operator-sdk edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

