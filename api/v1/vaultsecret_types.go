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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// +kubebuilder:validation:Required
	StringData map[string]string `json:"stringData"`
	// stringData allows specifying non-binary secret data in string form. It is
	// provided as a write-only input field for convenience. All keys and values
	// are merged into the data field on write, overwriting any existing values.
	// The stringData field is never output when reading from the API.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=Opaque
	Type string `json:"type,omitempty"`
}

// VaultSecretStatus defines the observed state of VaultSecret.
type VaultSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	SecretName string `json:"secretName,omitempty"`
	Message    string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.status.secretName`
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

const (
	AnnotationVaultMount           = "vault.hashicorp.com/mount"
	AnnotationVaultPath            = "vault.hashicorp.com/path"
	AnnotationVaultRefreshInterval = "vault.hashicorp.com/refresh-interval"
)

// VaultSecretAnnotation a list of VaultSecret.
type VaultSecretAnnotations struct {
	VaultPath            string `json:"vaultPath"`
	VaultMount           string `json:"vaultMount"`
	VaultRefreshInterval time.Duration `json:"vaultRefreshInterval"`
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

	if val, ok := ann[AnnotationVaultPath]; ok {
		annotations.VaultPath = val
	} else {
		return VaultSecretAnnotations{}, fmt.Errorf("missing required annotation: %s", AnnotationVaultPath)
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
