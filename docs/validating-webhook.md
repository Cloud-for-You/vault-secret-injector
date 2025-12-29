# Validating Webhook

Implementuje webhook pro Namespace zdroje, který automaticky spravuje Vault zdroje při životním cyklu Namespace. Tato funkce přináší uživatelům následující výhody:

- **Automatická izolace tajných údajů**: Každý Namespace získá své vlastní Vault politiky a role bez manuální konfigurace
- **Zabezpečení podle principu nejmenších privilegií**: Service Accounts v Namespace mohou přistupovat pouze k tajným údajům svého Namespace
- **Automatické čištění**: Při smazání Namespace se automaticky odstraní související Vault zdroje, čímž se předchází zanechání osiřelých politik
- **Auditovatelnost**: Všechny akce webhooku jsou auditovány ve Vault pro sledování změn
- **Zjednodušení správy**: Eliminuje potřebu manuální správy Vault politik pro každý Namespace

Webhook provádí následující akce při vytváření, aktualizaci nebo mazání Namespace:

- **Při vytváření Namespace**: Webhook se autentizuje do HashiCorp Vault pomocí Kubernetes autentizace (přes `/auth/kubernetes/login` API endpoint), vytvoří Vault politiku umožňující čtení a výpis tajných údajů v cestě `secret/data/{namespace}/*` (přes `sys/policies/acl` API), a vytvoří Kubernetes autentizační roli ve Vault vázanou na daný Namespace s touto politikou (přes `auth/kubernetes/role` API).
- **Při aktualizaci Namespace**: V současné době neprovádí žádné akce (vyhrazeno pro budoucí rozšíření).
- **Při mazání Namespace**: Webhook se autentizuje do Vault (přes `/auth/kubernetes/login`), smaže odpovídající Kubernetes autentizační roli (přes `auth/kubernetes/role/{namespace}` API) a politiku spojenou s Namespace (přes `sys/policies/acl/{namespace}-policy` API).

### Detaily nastavení vytvářeného webhookem

Webhook automaticky vytváří následující Vault zdroje pro každý nový Namespace:

1. **KV v2 Secret Engine**: Engine s názvem `kv-{namespace}` typu KV version 2 pro izolaci tajných údajů na úrovni namespace.

2. **Vault ACL Policy**: Politika s názvem `policy-{namespace}` obsahující pravidla:
   - `path "kv-{namespace}/data/*" { capabilities = ["read"] }`

3. **Kubernetes Auth Role**: Role s názvem `{namespace}` v `auth/{mount}/role` (kde mount je typicky `kubernetes` nebo `k8s-kind`) s konfigurací:
   - `bound_service_account_names`: `["*"]` (všechny service accounty v namespace)
   - `bound_service_account_namespaces`: `["{namespace}"]`
   - `token_policies`: `["policy-{namespace}"]`
   - `audience`: `"serviceaccount"`
   - `alias_name_source`: `"serviceaccount_uid"`
   - `token_type`: `"default"`
   - `token_ttl`: `"10m"`
   - `token_max_ttl`: `"10m"`
   - `token_no_default_policy`: `true`

Webhook používá následující Vault API endpointy pro správu zdrojů:
- **Autentizace**: `POST /auth/{mount}/login` (kde mount je typicky `kubernetes` nebo `k8s-kind`)
- **Vytvoření secret engine**: `POST /sys/mounts/kv-{namespace}`
- **Vytvoření politiky**: `PUT /sys/policies/acl/policy-{namespace}`
- **Vytvoření role**: `POST /auth/{mount}/role/{namespace}`
- **Smazání role**: `DELETE /auth/{mount}/role/{namespace}`
- **Smazání politiky**: `DELETE /sys/policies/acl/policy-{namespace}`
- **Smazání secret engine**: `DELETE /sys/mounts/kv-{namespace}`