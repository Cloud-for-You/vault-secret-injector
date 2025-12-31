package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KeyVault Secret engine specific annotations
	AnnotationVaultKVMount = "vault.hashicorp.com/mount" // Specify required mount point in Vault to fetch the secret from.
	AnnotationVaultKVPath  = "vault.hashicorp.com/path"  // Specify the path in Vault to fetch the secret from. If specified fetch all keys from this path. If not specified, fetch only data from keys defined in spec.stringData
	// Database Secret engine specific annotations
	AnnotationVaultDBMount = "vault.hashicorp.com/mount" // Specific database name or path in vault (e.g., db-my-dbname)
	AnnotationVaultDBRole  = "vault.hashicorp.com/role"  // Specific database role in vault to fetch credentials (e.g., my-username)

	// Global annotations
	AnnotationVaultRefreshInterval = "vault.hashicorp.com/refresh-interval" // Specify how often to refresh the secret from Vault.
	AnnotationVaultSecretName      = "vault.hashicorp.com/secret-name"      // Specify the name of the Kubernetes Secret to create/update.
	AnnotationVaultServiceAccount  = "vault.hashicorp.com/service-account"  // Specify the serviceaccount to use for Vault authentication.
)

// VaultSecretAnnotations a list of KeyVault.
type VaultSecretAnnotations struct {
	VaultKVMount string `json:"vaultKVMount"`
	VaultKVPath  string `json:"vaultKVPath"`
	VaultDBMount string `json:"vaultDBMount"`
	VaultDBRole  string `json:"vaultDBRole"`
	// Support for global settings
	VaultRefreshInterval time.Duration `json:"vaultRefreshInterval"`
	VaultSecretName      string        `json:"vaultSecretName"`
	VaultServiceAccount  string        `json:"vaultServiceAccount"`
}

func defaultAnnotations(meta *metav1.ObjectMeta) VaultSecretAnnotations {
	return VaultSecretAnnotations{
		VaultKVMount:         "kv-" + meta.GetNamespace(),
		VaultDBMount:         "db-" + meta.GetNamespace(),
		VaultDBRole:          meta.GetName(),
		VaultRefreshInterval: 5 * time.Minute,
		VaultServiceAccount:  "default",
	}
}

// GetAnnotations parses the annotations from the KeyVault object.
func ParseAnnotations(meta *metav1.ObjectMeta) (VaultSecretAnnotations, error) {
	annotations := defaultAnnotations(meta)
	ann := meta.GetAnnotations()

	if val, ok := ann[AnnotationVaultKVMount]; ok {
		annotations.VaultKVMount = val
	}

	if val, ok := ann[AnnotationVaultKVPath]; ok {
		annotations.VaultKVPath = val
	}

	if val, ok := ann[AnnotationVaultDBMount]; ok {
		annotations.VaultDBMount = val
	}

	if val, ok := ann[AnnotationVaultDBRole]; ok {
		annotations.VaultDBRole = val
	}

	if val, ok := ann[AnnotationVaultRefreshInterval]; ok {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return VaultSecretAnnotations{}, err
		}
		annotations.VaultRefreshInterval = duration
	}

	if val, ok := ann[AnnotationVaultSecretName]; ok {
		annotations.VaultSecretName = val
	} else {
		annotations.VaultSecretName = meta.Name
	}

	if val, ok := ann[AnnotationVaultServiceAccount]; ok {
		annotations.VaultServiceAccount = val
	}

	return annotations, nil
}
