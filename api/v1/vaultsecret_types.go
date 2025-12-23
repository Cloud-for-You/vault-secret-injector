/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VaultSecretSpec defines the desired state of VaultSecret.
type VaultSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Immutable, if set to true, ensures that data stored in the Secret cannot be
	// updated (only object metadata can be modified). If not set to true, the
	// field can be modified at any time. Defaulted to nil.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Immutable bool `json:"immutable,omitempty"`
	// StringData is an example field of VaultSecret. Edit vaultsecret_types.go to remove/update
	// +kubebuilder:validation:Optional
	StringData map[string]string `json:"stringData"`
	// stringData allows specifying non-binary secret data in string form. It is
	// provided as a write-only input field for convenience. All keys and values
	// are merged into the data field on write, overwriting any existing values.
	// The stringData field is never output when reading from the API.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=Opaque
	Type string `json:"type,omitempty"`
	// list of other CRD object for rollout/restart purposes
	// +kubebuilder:validation:Optional
	RestartObjectsRef []RestartOjectRef `json:"restartObjectsRef,omitempty"`
}

// VaultSecretStatus defines the observed state of VaultSecret.
type VaultSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	SecretName  string `json:"secretName,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`
	Message     string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.status.secretName`
// +kubebuilder:printcolumn:name="Last Updated",type=string,JSONPath=`.status.lastUpdated`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`

// VaultSecret is the Schema for the vaultsecrets API.
type VaultSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultSecretSpec   `json:"spec,omitempty"`
	Status VaultSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VaultSecretList contains a list of VaultSecret.
type VaultSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultSecret{}, &VaultSecretList{})
}

type RestartOjectRef struct {
	// +kubebuilder:validation:Enum=apps/v1
	APIVersion string `json:"apiVersion"`
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

const (
	// Specify required mount point in Vault to fetch the secret from.
	AnnotationVaultMount = "vault.hashicorp.com/mount"
	// Specify the path in Vault to fetch the secret from.
	// If specified fetch all keys from this path.
	// If not specified, fetch only data from keys defined in spec.stringData
	AnnotationVaultPath = "vault.hashicorp.com/path"
	// Specify how often to refresh the secret from Vault.
	AnnotationVaultRefreshInterval = "vault.hashicorp.com/refresh-interval"
	// Specify the name of the Kubernetes Secret to create/update.
	AnnotationVaultSecretName = "vault.hashicorp.com/secret-name"
)

// VaultSecretAnnotation a list of VaultSecret.
type VaultSecretAnnotations struct {
	VaultPath            string        `json:"vaultPath"`
	VaultMount           string        `json:"vaultMount"`
	VaultRefreshInterval time.Duration `json:"vaultRefreshInterval"`
	VaultSecretName      string        `json:"vaultSecretName"`
}

func defaultAnnotations(namespace string) VaultSecretAnnotations {
	return VaultSecretAnnotations{
		VaultMount:           "kv-" + namespace,
		VaultRefreshInterval: 5 * time.Minute,
	}
}

// GetAnnotations parses the annotations from the VaultSecret object.
func (vs *VaultSecret) ParseAnnotations(meta metav1.ObjectMeta) (VaultSecretAnnotations, error) {
	annotations := defaultAnnotations(meta.Namespace)
	ann := meta.GetAnnotations()

	if val, ok := ann[AnnotationVaultMount]; ok {
		annotations.VaultMount = val
	}

	if val, ok := ann[AnnotationVaultSecretName]; ok {
		annotations.VaultSecretName = val
	} else {
		annotations.VaultSecretName = meta.Name
	}

	if val, ok := ann[AnnotationVaultPath]; ok {
		annotations.VaultPath = val
	}

	if val, ok := ann[AnnotationVaultRefreshInterval]; ok {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return VaultSecretAnnotations{}, err
		}
		annotations.VaultRefreshInterval = duration
	}

	return annotations, nil
}

func (vs *VaultSecret) CreateOrUpdateK8sSecret(ctx context.Context, c client.Client, secretData map[string][]byte) error {
	annotations, err := vs.ParseAnnotations(vs.ObjectMeta)
	if err != nil {
		return err
	}
	secretName := annotations.VaultSecretName
	secretNamespace := vs.Namespace

	// Add ownereship reference
	ownerRef := metav1.NewControllerRef(vs, vs.GroupVersionKind())

	k8sSecret := &corev1.Secret{}
	err = c.Get(ctx, client.ObjectKey{Name: secretName, Namespace: secretNamespace}, k8sSecret)
	if err != nil {
		// Secret does not exist, create it
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: secretNamespace,
				OwnerReferences: []metav1.OwnerReference{
					*ownerRef,
				},
			},
			Data:      secretData,
			Type:      corev1.SecretType(vs.Spec.Type),
			Immutable: &vs.Spec.Immutable,
		}
		if err := c.Create(ctx, newSecret); err != nil {
			return err
		}
		vs.Status.SecretName = secretName
		return nil
	}

	// Secret exists, update it
	k8sSecret.Data = secretData
	k8sSecret.Type = corev1.SecretType(vs.Spec.Type)
	k8sSecret.Immutable = &vs.Spec.Immutable

	if err := c.Update(ctx, k8sSecret); err != nil {
		return err
	}
	vs.Status.SecretName = secretName
	return nil
}
