package vault

import (
	"context"
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	vaultk8s "github.com/hashicorp/vault/api/auth/kubernetes"
)

// VaultLoginWithK8sAuth performs login to Vault using Kubernetes authentication.
// Parameters:
// - ctx: context for the login request
// - client: the Vault API client
// - mount: the mount path for Kubernetes auth
// - jwt: the JWT token for authentication
// - role: the Vault role to authenticate as
// Returns: error if login fails
func VaultLoginWithK8sAuth(ctx context.Context, client *vaultapi.Client, mount string, jwt string, role string) error {
	auth, err := vaultk8s.NewKubernetesAuth(
		role,
		vaultk8s.WithServiceAccountToken(jwt),
		vaultk8s.WithMountPath(mount),
	)
	if err != nil {
		return err
	}
	sec, err := client.Auth().Login(ctx, auth)
	if err != nil {
		return err
	}
	if sec == nil {
		return fmt.Errorf("no secret returned from login")
	}

	LogAudit(jwt, "Vault login with Kubernetes auth", map[string]interface{}{})

	return nil
}
