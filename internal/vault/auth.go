package vault

import (
	"context"
	"fmt"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	vaultk8s "github.com/hashicorp/vault/api/auth/kubernetes"
)

func RequestSAToken(ctx context.Context, namespace, serviceaccount, audience string, ttl int64) (string, error) {
	// TODO: Implement this function
	return "", nil
}

func NewVaultClient() (*vaultapi.Client, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = os.Getenv("VAULT_ADDR")
	if os.Getenv("VAULT_TLS_SKIP_VERIFY") == "true" {
		cfg.ConfigureTLS(&vaultapi.TLSConfig{Insecure: true})
	}

	return vaultapi.NewClient(cfg)
}

func VaultLoginWithK8sAuth(ctx context.Context, client *vaultapi.Client, mount string, jwt []byte, role string) error {
  auth, err := vaultk8s.NewKubernetesAuth(
    role,
    vaultk8s.WithServiceAccountToken(string(jwt)),
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

	return nil
}