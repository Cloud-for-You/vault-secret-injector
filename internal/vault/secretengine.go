package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

// CreateSecretEngineKV2 creates a KV v2 secret engine mount in Vault for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if creation fails
func CreateSecretEngineKV(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("kv-%s", ctx.GetName())
	
	mountInput := &vaultapi.MountInput{
		Type: "kv",
		Options: map[string]string{
			"version": "2",
		},
	}
	
	err := client.Sys().Mount(mountPath, mountInput)
	if err != nil {
		return fmt.Errorf("failed to create secret engine at %s: %w", mountPath, err)
	}

	LogAudit(jwt, "Created Vault secret engine", map[string]interface{}{"mountPath": mountPath, "namespace": ctx.GetName()})

	return nil
}

// DeleteSecretEngineKV2 deletes the KV v2 secret engine mount in Vault for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if deletion fails
func DeleteSecretEngineKV(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("secret-%s", ctx.GetName())

	err := client.Sys().Unmount(mountPath)
	if err != nil {
		return fmt.Errorf("failed to delete secret engine at %s: %w", mountPath, err)
	}

	LogAudit(jwt, "Deleted Vault secret engine", map[string]interface{}{"mountPath": mountPath, "namespace": ctx.GetName()})

	return nil
}

// FetchSecretKV fetches a secret from Vault KV v2 engine.
// Parameters:
// - client: the Vault API client
// - jwt: the JWT token for auditing
// - mount: the mount path of the KV engine
// - path: the path to the secret in Vault
// Returns: the secret data as a map or error
func FetchSecretEngineKV(client *vaultapi.Client, jwt, mount, path string) (map[string]interface{}, error) {
	secretPath := fmt.Sprintf("%s/data/%s", mount, path)
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", secretPath, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at %s", secretPath)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format at %s", secretPath)
	}

	LogAudit(jwt, "Fetched secret from Vault KV engine", map[string]interface{}{"mount": mount, "path": path})

	return data, nil
}