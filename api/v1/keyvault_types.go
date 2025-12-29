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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeyVaultSpec defines the desired state of KeyVault.
type KeyVaultSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Immutable, if set to true, ensures that data stored in the Secret cannot be
	// updated (only object metadata can be modified). If not set to true, the
	// field can be modified at any time. Defaulted to nil.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Immutable bool `json:"immutable,omitempty"`

	// StringData is an example field of KeyVault. Edit keyvault_types.go to remove/update.
	// StringData allows specifying non-binary secret data in string form. It is
	// provided as a write-only input field for convenience. All keys and values
	// are merged into the data field on write, overwriting any existing values.
	// The stringData field is never output when reading from the API.
	// +kubebuilder:validation:Optional
	StringData map[string]KeyVaultDataValue `json:"stringData,omitempty"`

	// Type of the secret, e.g. Opaque, kubernetes.io/dockerconfigjson, etc.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=Opaque
	Type string `json:"type,omitempty"`

	// List of other CRD object for rollout/restart purposes
	// +kubebuilder:validation:Optional
	RolloutObjectRef []RolloutObjectRef `json:"rolloutObjectsRef,omitempty"`
}

// KeyVaultStatus defines the observed state of KeyVault.
type KeyVaultStatus struct {
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

// KeyVault is the Schema for the keyvaults API.
type KeyVault struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyVaultSpec   `json:"spec,omitempty"`
	Status KeyVaultStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyVaultList contains a list of KeyVault.
type KeyVaultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeyVault `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeyVault{}, &KeyVaultList{})
}
