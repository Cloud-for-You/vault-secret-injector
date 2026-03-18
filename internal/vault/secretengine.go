package vault

import (
	"fmt"
	"strings"

	cfyczv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

// CreateOrUpdateSecretEngineKV creates or updates a KV v2 secret engine mount in Vault for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if creation or update fails
func CreateOrUpdateSecretEngineKV(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	mountPath := fmt.Sprintf("kv-%s", ctx.GetName())

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	if _, exists := mounts[mountPath+"/"]; exists {
		LogAudit(jwt, "Vault secret engine already exists", map[string]interface{}{"mountPath": mountPath, "namespace": ctx.GetName()})
		return nil
	}

	mountInput := &vaultapi.MountInput{
		Type: "kv",
		Description: fmt.Sprintf(
			"Managed KV v2 (project=%s;owner=vault-secret-injector-webhook;purpose=secrets;lifecycle=managed)", ctx.GetName(),
		),
		Options: map[string]string{
			"version": "2",
		},
	}

	err = client.Sys().Mount(mountPath, mountInput)
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
	mountPath := fmt.Sprintf("kv-%s", ctx.GetName())

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
// Returns: the secret data as map[string][]byte or error
func FetchSecretEngineKV(client *vaultapi.Client, jwt, mount, path string) (map[string][]byte, error) {
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

	result := make(map[string][]byte)
	for k, v := range data {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value for key %s is not a string", k)
		}
		result[k] = []byte(str)
	}

	LogAudit(jwt, "Fetched secret from Vault KV engine", map[string]interface{}{"mount": mount, "path": path})

	return result, nil
}

// FetchSecretValue fetches a single value from Vault KV v2 engine at the given path.
// Assumes the secret has at least one key-value pair and returns the value of the first key.
func FetchSecretValue(client *vaultapi.Client, jwt, mount, path string) ([]byte, error) {
	data, err := FetchSecretEngineKV(client, jwt, mount, path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no data found at path %s", path)
	}
	// Return the value of the first key
	for _, v := range data {
		return v, nil
	}
	return nil, nil // shouldn't reach
}

// FetchSecretData fetches secret data from Vault based on annotations and spec.
func FetchSecretData(vaultClient *vaultapi.Client, impersonateJwt string, annotations *cfyczv1.KeyVaultAnnotations, vaultSecret *cfyczv1.KeyVault) (map[string][]byte, error) {
	var secretData map[string][]byte
	if annotations.VaultPath != "" {
		data, err := FetchSecretEngineKV(vaultClient, impersonateJwt, annotations.VaultMount, annotations.VaultPath)
		if err != nil {
			return nil, err
		}
		secretData = data
	} else {
		secretData = make(map[string][]byte)
		for secretKey, vaultSpec := range vaultSecret.Spec.StringData {
			parts := strings.Split(string(vaultSpec), "@")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid stringData format for key %s: expected <vaultPath>@<key>", secretKey)
			}
			vaultPath := parts[0]
			keyInVault := parts[1]
			data, err := FetchSecretEngineKV(vaultClient, impersonateJwt, annotations.VaultMount, vaultPath)
			if err != nil {
				return nil, err
			}
			value, ok := data[keyInVault]
			if !ok {
				return nil, fmt.Errorf("key %s not found in vault path %s", keyInVault, vaultPath)
			}
			secretData[secretKey] = value
		}
	}
	return secretData, nil
}
