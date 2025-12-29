# Správa CR Database

## Reconciliation Custom CRD

Operator implementuje reconciliation smyčku pro Database CRD, která zajišťuje kontinuální správu databázových přihlašovacích údajů z Vault do Kubernetes Secrets.

### Autentizace do Vault

Pro získávání databázových přihlašovacích údajů z Vault je vždy využíván impersonovaný JWT token z namespace, kde je Database CRD nasazen. Ve výchozím nastavení se používá ServiceAccount `default` z tohoto namespace. JWT token tohoto ServiceAccount je impersonován pro autentizaci do Vault a získání tokenu s příslušnými oprávněními.

Pouze na základě specifického požadavku je možné přetížit použitý ServiceAccount pomocí anotace `vault.hashicorp.com/service-account`, což umožňuje použít specifický ServiceAccount s vlastními oprávněními nebo rolí.

### Typy databázových přihlašovacích údajů

Operator podporuje dva typy databázových přihlašovacích údajů z Vault:

1. **Dynamic credentials**:
   - Vault generuje nové přihlašovací údaje při každém požadavku.
   - Přihlašovací údaje jsou dočasné a automaticky se ruší po uplynutí času.
   - Ideální pro zvýšení bezpečnosti, protože zabraňuje opakovanému použití stejných credentials.

2. **Static credentials**:
   - Používají se pevně definované přihlašovací údaje uložené ve Vault.
   - Credentials jsou trvalé a nemění se.
   - Vhodné pro scénáře, kde je potřeba stabilní přístup.

   **Proměnné pro templating:**
   - `{{ .username }}`: Uživatelské jméno
   - `{{ .password }}`: Heslo
   - `{{ .last_vault_rotation }}`: Čas poslední rotace v Vault (ISO 8601 formát)
   - `{{ .rotation_period }}`: Perioda rotace v sekundách
   - `{{ .ttl }}`: Time-to-live v sekundách

Typ credentials se specifikuje v poli `spec.credsType` jako `dynamic` nebo `static`.

### Šablona připojovacího řetězce

Pole `spec.stringTemplate` je mapa řetězců (`map[string]string`), kde každý klíč představuje název klíče v Kubernetes Secret a hodnota je šablona řetězce s placeholdery, které se nahradí skutečnými hodnotami z Vault.

Příklad šablony:

```yaml
stringTemplate:
  connectionString: "postgresql://{{ .username }}:{{ .password }}@db.example.com:5432/mydb"
  username: "{{ .username }}"
```

Operator nahradí `{{ .username }}`, `{{ .password }}` atd. hodnotami získanými z Vault a vytvoří Secret s odpovídajícími klíči.

### Příklady použití

**Dynamic credentials:**
```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: Database
metadata:
  name: database-dynamic
spec:
  credsType: dynamic
  stringTemplate:
    connectionString: "postgresql://{{ .username }}:{{ .password }}@db.example.com:5432/mydb"
```

**Static credentials:**
```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: Database
metadata:
  name: database-static
spec:
  credsType: static
  stringTemplate:
    connectionString: "mysql://{{ .username }}:{{ .password }}@db.example.com:3306/mydb"
```

**Podporované anotace:**
- `vault.hashicorp.com/mount`: Určuje mount point ve Vault (výchozí: `database`)
- `vault.hashicorp.com/role`: Název role pro databázové credentials (používá se pro dynamic creds)
- `vault.hashicorp.com/refresh-interval`: Interval pro automatické obnovení dat (výchozí: `5 minut`)
- `vault.hashicorp.com/secret-name`: Název vytvořeného Kubernetes Secret (výchozí: `název Database`)
- `vault.hashicorp.com/service-account`: Jméno ServiceAccount, které se použije pro impersonate JWT tokenu (výchozí: `default`)

## Automatický rollout aplikací

Automaticky spoustí rollout (restart) aplikací při aktualizaci databázových přihlašovacích údajů do objektu Secret. To zajišťuje, že aplikace okamžitě použijí nové credentials bez manuálního zásahu.

### Konfigurace rollout objektů

V `spec.rolloutObjectsRef` můžete specifikovat seznam Kubernetes objektů, které se mají restartovat při změně dat:

```yaml
apiVersion: vaultsecret.cfy.cz/v1
kind: Database
metadata:
  name: my-database
spec:
  credsType: dynamic
  stringTemplate:
    connectionString: "postgresql://{{ .username }}:{{ .password }}@db.example.com:5432/mydb"
  rolloutObjectsRef:
  - apiVersion: apps/v1
    kind: Deployment
    name: my-app-deployment
```

**Podporované typy objektů:**
- `Deployment`
- `StatefulSet`
- `DaemonSet`

Rollout se spustí pouze pokud se data v Secret skutečně změnila. Operator aktualizuje anotace v `spec.template.metadata.annotations` s časovým razítkem, což vyvolá rolling update.