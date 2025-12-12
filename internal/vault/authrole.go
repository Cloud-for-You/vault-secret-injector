package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

// CreateVaultKubernetesAuthRole creates a Kubernetes auth role in Vault for the given namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - mount: the mount path for Kubernetes auth
// - jwt: the JWT token for auditing
// Returns: error if creation fails
func CreateVaultKubernetesAuthRole(ctx *k8siov1.Namespace, client *vaultapi.Client, mount string, jwt string) error {
	roleName := ctx.GetName()
	policyName := fmt.Sprintf("%s-policy", ctx.GetName())

	roleData := map[string]interface{}{
		"bound_service_account_names":      []string{"*"},
		"bound_service_account_namespaces": []string{ctx.GetName()},
		"policies":                         []string{policyName},
	}

	path := fmt.Sprintf("auth/%s/role/%s", mount, roleName)
	_, err := client.Logical().Write(path, roleData)
	if err != nil {
		return fmt.Errorf("failed to create Vault role %s: %w", roleName, err)
	}

	LogAudit(jwt, "Created Vault Kubernetes auth role", map[string]interface{}{"roleName": roleName, "roleData": roleData})

	return nil
}

// DeleteVaultKubernetesAuthRole deletes the Kubernetes auth role in Vault for the given namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - mount: the mount path for Kubernetes auth
// - jwt: the JWT token for auditing
// Returns: error if deletion fails
func DeleteVaultKubernetesAuthRole(ctx *k8siov1.Namespace, client *vaultapi.Client, mount string, jwt string) error {
	roleName := ctx.GetName()

	path := fmt.Sprintf("auth/%s/role/%s", mount, roleName)
	_, err := client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("failed to delete Vault role %s: %w", roleName, err)
	}

	LogAudit(jwt, "Deleted Vault Kubernetes auth role", map[string]interface{}{"roleName": roleName, "namespace": ctx.GetName()})

	return nil
}