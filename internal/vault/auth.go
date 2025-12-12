package vault

import (
	"context"
	"fmt"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	vaultk8s "github.com/hashicorp/vault/api/auth/kubernetes"
)

// NewVaultClient creates a new Vault API client with configuration from environment variables.
// Uses VAULT_ADDR and VAULT_TLS_SKIP_VERIFY environment variables.
// Returns: the configured Vault client or error
func NewVaultClient() (*vaultapi.Client, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = os.Getenv("VAULT_ADDR")
	if os.Getenv("VAULT_TLS_SKIP_VERIFY") == "true" {
		cfg.ConfigureTLS(&vaultapi.TLSConfig{Insecure: true})
	}

	return vaultapi.NewClient(cfg)
}

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