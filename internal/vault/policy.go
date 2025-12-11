package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

func CreateVaultPolicy(ctx *k8siov1.Namespace, client *vaultapi.Client) error {
	policyName := fmt.Sprintf("%s-policy", ctx.GetName())
	policyRules := fmt.Sprintf(`
path "secret/data/%s/*" { capabilities = ["read","list"] }
`, ctx.GetName())

	err := client.Sys().PutPolicy(policyName, policyRules)
	if err != nil {
		return fmt.Errorf("failed to create policy %s: %w", policyName, err)
	}

	return nil
}
