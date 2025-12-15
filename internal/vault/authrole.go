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
	policyName := fmt.Sprintf("policy-%s", ctx.GetName())

	roleData := map[string]interface{}{
		// Kubernetes auth role configuration
		"alias_name_source":                        "serviceaccount_uid",
		"audience":                                 "serviceaccount",
		"bound_service_account_names":              []string{"*"},
		"bound_service_account_namespaces":         []string{ctx.GetName()},
		"bound_service_account_namespace_selector": "",
		// Vault policies to attach to tokens issued for this role
		"token_type":              "default",
		"token_ttl":               "10m",
		"token_max_ttl":           "10m",
		"token_explicit_max_ttl":  "10m",
		"token_policies":          []string{policyName},
		"token_no_default_policy": true,
		"token_num_uses":          0,
		"token_period":            0,
		"token_bound_cidrs":       []string{},
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
