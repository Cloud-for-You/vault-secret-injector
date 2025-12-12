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
func CreateSecretEngineKV2(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
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
func DeleteSecretEngineKV2(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("secret-%s", ctx.GetName())

	err := client.Sys().Unmount(mountPath)
	if err != nil {
		return fmt.Errorf("failed to delete secret engine at %s: %w", mountPath, err)
	}

	LogAudit(jwt, "Deleted Vault secret engine", map[string]interface{}{"mountPath": mountPath, "namespace": ctx.GetName()})

	return nil
}