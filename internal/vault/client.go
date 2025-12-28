package vault

import (
	"context"
	"fmt"
	"os"

	cfyczv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
	vaultapi "github.com/hashicorp/vault/api"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// getImpersonateSAToken requests a service account token.
// Parameters:
// - ctx: context for the request
// - clientset: the Kubernetes clientset
// - namespace: the namespace
// - serviceaccount: the service account name
// - audience: the audience for the token
// - ttl: time to live in seconds
// Returns: the JWT token string or error
func getImpersonateSAToken(ctx context.Context, clientset *kubernetes.Clientset, namespace, serviceAccount, audience string, ttl int64) (string, error) {
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{audience},
			ExpirationSeconds: &ttl,
		},
	}

	result, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, serviceAccount, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Status.Token, nil
}

// NewVaultClient creates a new Vault API client with configuration from environment variables.
// Uses VAULT_ADDR and VAULT_TLS_SKIP_VERIFY environment variables.
// Returns: the configured Vault client or error
func NewVaultClient() (*vaultapi.Client, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = os.Getenv("VAULT_ADDR")
	if os.Getenv("VAULT_TLS_SKIP_VERIFY") == "true" {
		if err := cfg.ConfigureTLS(&vaultapi.TLSConfig{Insecure: true}); err != nil {
			return nil, fmt.Errorf("configure vault TLS: %w", err)
		}
	}

	return vaultapi.NewClient(cfg)
}

// SetupVaultClient sets up the Vault client and authenticates.
func SetupVaultClient(ctx context.Context, vaultSecret *cfyczv1.KeyVault) (*vaultapi.Client, string, error) {
	annotations, err := vaultSecret.ParseAnnotations(vaultSecret.ObjectMeta)
	if err != nil {
		return nil, "", err
	}
	clientset := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	impersonateJwt, err := getImpersonateSAToken(ctx, clientset, vaultSecret.GetNamespace(), annotations.VaultServiceAccount, "serviceaccount", int64(600))
	if err != nil {
		return nil, "", err
	}
	LogAudit(impersonateJwt, "Obtained impersonated service account token", map[string]interface{}{"namespace": vaultSecret.GetNamespace(), "serviceAccount": annotations.VaultServiceAccount})

	vaultClient, err := NewVaultClient()
	if err != nil {
		return nil, "", err
	}

	err = VaultLoginWithK8sAuth(ctx, vaultClient, os.Getenv("VAULT_K8S_AUTH_MOUNT"), impersonateJwt, vaultSecret.GetNamespace())
	if err != nil {
		return nil, "", err
	}
	return vaultClient, impersonateJwt, nil
}
