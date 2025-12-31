package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

// CreateOrUpdateSecretEngineDatabase creates or updates a database secret engine mount in Vault for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if creation or update fails
func CreateOrUpdateSecretEngineDatabase(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("db-%s", ctx.GetName())

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	if _, exists := mounts[mountPath+"/"]; exists {
		LogAudit(jwt, "Database secret engine already exists", map[string]any{"mountPath": mountPath, "namespace": ctx.GetName()})
		return nil
	}

	mountInput := &vaultapi.MountInput{
		Type: "database",
	}

	err = client.Sys().Mount(mountPath, mountInput)
	if err != nil {
		return fmt.Errorf("failed to create Database secret engine at %s: %w", mountPath, err)
	}

	LogAudit(jwt, "Created Database secret engine", map[string]any{"mountPath": mountPath, "namespace": ctx.GetName()})

	return nil
}

// DeleteSecretEngineDatabase deletes the database secret engine mount in Vault for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if deletion fails
func DeleteSecretEngineDatabase(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("db-%s", ctx.GetName())

	err := client.Sys().Unmount(mountPath)
	if err != nil {
		return fmt.Errorf("failed to delete Database secret engine at %s: %w", mountPath, err)
	}

	LogAudit(jwt, "Deleted Database secret engine", map[string]any{"mountPath": mountPath, "namespace": ctx.GetName()})

	return nil
}

// FetchDatabaseSecret fetches database credentials from Vault database engine.
// Parameters:
// - client: the Vault API client
// - jwt: the JWT token for auditing
// - database: the mount path of the database engine
// - path: the path to the database role (e.g., static-creds/static)
// Returns: the secret data as map[string][]byte or error
func FetchSecretEngineDatabase(client *vaultapi.Client, jwt, database, role string) (*vaultapi.Secret, error) {
	secretPath := fmt.Sprintf("%s/%s", database, role)
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read database secret at %s: %w", secretPath, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at %s", secretPath)
	}

	LogAudit(jwt, "Fetched database secret from Vault", map[string]any{"database": database, "role": role})

	return secret, nil
}
