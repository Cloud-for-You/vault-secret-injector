# Správa CR KeyVault

## Reconciliation Custom CRD

Operator implementuje reconciliation smyčku pro KeyVault CRD, která zajišťuje kontinuální synchronizaci tajných údajů z Vault do Kubernetes Secrets.

### Autentizace do Vault

Pro získávání tajemství z Vault je vždy využíván impersonovaný JWT token z namespace, kde je KeyVault CRD nasazen. Ve výchozím nastavení se používá ServiceAccount `default` z tohoto namespace. JWT token tohoto ServiceAccount je impersonován pro autentizaci do Vault a získání tokenu s příslušnými oprávněními.

Pouze na základě specifického požadavku je možné přetížit použitý ServiceAccount pomocí anotace `vault.hashicorp.com/service-account`, což umožňuje použít specifický ServiceAccount s vlastními oprávněními nebo rolí.

### Možnosti přístupu k citlivým datům

Operator podporuje dva způsoby přístupu k tajným údajům z Vault:

1. **Načtení všech klíčů z cesty (path annotation)**:
   - Použijte anotaci `vault.hashicorp.com/path` s cestou ve Vault.
   - Operator načte všechny klíče z této cesty a vytvoří Kubernetes Secret s odpovídajícími daty.
   - Příklad: Načtení všech konfigurací z cesty `myapp/config`.

2. **Načtení specifických klíčů (stringData)**:
   - Použijte pole `spec.stringData` ve formátu `data_key: <vault_path>@<key_in_vault>`.
   - Operator načte hodnotu specifického klíče z dané cesty ve Vault a uloží ji do Kubernetes Secret pod zadaným klíčem.
   - Příklad: Načtení hesla z `myapp1/config` pod klíčem `DATA1` a z `myapp2/config` pod klíčem `DATA2`.

Musí být definována buď anotace `vault.hashicorp.com/path`, nebo `spec.stringData`. Nelze kombinovat obě metody v jednom KeyVault. Annotace má přednost před specifikovanými klíči.

### Příklady použití

**Načtení všech klíčů z cesty:**
```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: KeyVault
metadata:
  name: keyvault-all-keys
  annotations:
    vault.hashicorp.com/path: "myapp/config"
spec:
  type: Opaque
```

**Načtení specifických klíčů:**
```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: KeyVault
metadata:
  name: keyvault-specific-keys
spec:
  stringData:
    USERNAME: "myapp1/config@DATA1"
    PASSWORD: "myapp2/config@DATA2"
  type: Opaque
```

**Další podporované anotace:**
- `vault.hashicorp.com/mount`: Určuje mount point ve Vault (výchozí: `kv-{namespace_name}`)
- `vault.hashicorp.com/refresh-interval`: Interval pro automatické obnovení dat (výchozí: `5 minut`)
- `vault.hashicorp.com/secret-name`: Název vytvořeného Kubernetes Secret (výchozí: `název KeyVault`)
- `vault.hashicorp.com/service-account`: Jméno ServiceAccount, které se použije pro impersonate JWT tokenu (výchozí: `default`)

## Automatický rollout aplikací

Automaticky spoustí rollout (restart) aplikací při aktualizaci tajných údajů do objektu Secret. To zajišťuje, že aplikace okamžitě použijí nové hodnoty bez manuálního zásahu.

### Konfigurace rollout objektů

V `spec.rolloutObjectsRef` můžete specifikovat seznam Kubernetes objektů, které se mají restartovat při změně dat:

```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: KeyVault
metadata:
  name: my-secret
spec:
  stringData:
    PASSWORD: "myapp/config@password"
  rolloutObjectsRef:
  - apiVersion: apps/v1
    kind: Deployment
    name: my-deployment
  type: Opaque
```

**Podporované typy objektů:**
- `Deployment`
- `StatefulSet`
- `DaemonSet`

Rollout se spustí pouze pokud se data v Secret skutečně změnila. Operator aktualizuje anotace v `spec.template.metadata.annotations` s časovým razítkem, což vyvolá rolling update.