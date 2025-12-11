package vault

import (
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	k8siov1 "k8s.io/api/core/v1"
)

func CreateVaultKubernetesAuthRole(ctx *k8siov1.Namespace, client *vaultapi.Client, mount string) error {
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

	return nil
}