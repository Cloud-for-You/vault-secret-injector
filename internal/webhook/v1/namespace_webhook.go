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
	"fmt"
	"os"

	k8siov1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	vaultlib "github.com/cloud-for-you/vault-secret-injector/internal/vault"
)

// nolint:unused
// log is for logging in this package.
var namespacelog = logf.Log.WithName("namespace-resource")

// SetupNamespaceWebhookWithManager registers the webhook for Namespace in the manager.
func SetupNamespaceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&k8siov1.Namespace{}).
		WithValidator(&NamespaceCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-namespace,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update;delete,versions=v1,name=vnamespace-v1.kb.io,admissionReviewVersions=v1

// NamespaceCustomValidator struct is responsible for validating the Namespace resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NamespaceCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &NamespaceCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Namespace.
func (v *NamespaceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespace, ok := obj.(*k8siov1.Namespace)
	if !ok {
		return nil, fmt.Errorf("expected a Namespace object but got %T", obj)
	}
	namespacelog.Info("Validation for Namespace upon creation", "name", namespace.GetName())

  jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
  if err != nil {
    namespacelog.Error(err, "Failed to read ServiceAccount JWT token")
    return nil, err
  }
	role := os.Getenv("VAULT_ROLE")
	mount := os.Getenv("VAULT_K8S_AUTH_MOUNT")
	if mount == "" {
	  mount = "kubernetes"
	}

  vaultClient, err := vaultlib.NewVaultClient()
  if err != nil {
    namespacelog.Error(err, "Failed to create Vault client")
    return nil, err
  }

	err = vaultlib.VaultLoginWithK8sAuth(ctx, vaultClient, mount, string(jwt), role)
	if err != nil {
		namespacelog.Error(err, "Failed to login to Vault")
		return nil, err
	}

	err = vaultlib.CreateVaultPolicy(namespace, vaultClient, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to create Vault policy")
		return nil, err
	}

	err = vaultlib.CreateVaultKubernetesAuthRole(namespace, vaultClient, mount, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to create Vault Kubernetes auth role")
		return nil, err
	}

	err = vaultlib.CreateOrUpdateSecretEngineKV(namespace, vaultClient, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to create or update Vault secret engine")
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Namespace.
func (v *NamespaceCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	namespace, ok := newObj.(*k8siov1.Namespace)
	if !ok {
		return nil, fmt.Errorf("expected a Namespace object for the newObj but got %T", newObj)
	}
	namespacelog.Info("Validation for Namespace upon update", "name", namespace.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Namespace.
func (v *NamespaceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespace, ok := obj.(*k8siov1.Namespace)
	if !ok {
		return nil, fmt.Errorf("expected a Namespace object but got %T", obj)
	}
	namespacelog.Info("Validation for Namespace upon deletion", "name", namespace.GetName())

	jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		namespacelog.Error(err, "Failed to read ServiceAccount JWT token")
		return nil, err
	}
	role := os.Getenv("VAULT_ROLE")
	mount := os.Getenv("VAULT_K8S_AUTH_MOUNT")
	if mount == "" {
		mount = "kubernetes"
	}

	vaultClient, err := vaultlib.NewVaultClient()
	if err != nil {
		namespacelog.Error(err, "Failed to create Vault client")
		return nil, err
	}

	err = vaultlib.VaultLoginWithK8sAuth(ctx, vaultClient, mount, string(jwt), role)
	if err != nil {
		namespacelog.Error(err, "Failed to login to Vault")
		return nil, err
	}

	err = vaultlib.DeleteVaultKubernetesAuthRole(namespace, vaultClient, mount, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to delete Vault Kubernetes auth role")
		return nil, err
	}

	err = vaultlib.DeleteVaultPolicy(namespace, vaultClient, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to delete Vault policy")
		return nil, err
	}

	err = vaultlib.DeleteSecretEngineKV(namespace, vaultClient, string(jwt))
	if err != nil {
		namespacelog.Error(err, "Failed to delete Vault secret engine")
		return nil, err
	}

	return nil, nil
}
