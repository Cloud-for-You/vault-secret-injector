package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

// CreateVaultPolicy creates a Vault policy allowing read access to secrets in the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if creation fails
func CreateVaultPolicy(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	policyName := fmt.Sprintf("policy-%s", ctx.GetName())
	policyRules := fmt.Sprintf(`path "kv-%s/data/*" { capabilities = ["read"] }`, ctx.GetName())

	err := client.Sys().PutPolicy(policyName, policyRules)
	if err != nil {
		return fmt.Errorf("failed to create policy %s: %w", policyName, err)
	}

	LogAudit(jwt, "Created Vault policy", map[string]interface{}{"policyName": policyName, "policyRules": policyRules})

	return nil
}

// DeleteVaultPolicy deletes the Vault policy for the namespace.
// Parameters:
// - ctx: the Kubernetes namespace object
// - client: the Vault API client
// - jwt: the JWT token for auditing
// Returns: error if deletion fails
func DeleteVaultPolicy(ctx *k8siov1.Namespace, client *vaultapi.Client, jwt string) error {
	policyName := fmt.Sprintf("policy-%s", ctx.GetName())

	err := client.Sys().DeletePolicy(policyName)
	if err != nil {
		return fmt.Errorf("failed to delete policy %s: %w", policyName, err)
	}

	LogAudit(jwt, "Deleted Vault policy", map[string]interface{}{"policyName": policyName, "namespace": ctx.GetName()})

	return nil
}
