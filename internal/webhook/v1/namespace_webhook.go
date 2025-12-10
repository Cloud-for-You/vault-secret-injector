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
	"time"

	k8siov1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

// nolint:unused
// log is for logging in this package.
var namespacelog = logf.Log.WithName("namespace-resource")

var vaultClient *vault.Client
var err error

// SetupNamespaceWebhookWithManager registers the webhook for Namespace in the manager.
func SetupNamespaceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&k8siov1.Namespace{}).
		WithValidator(&NamespaceCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-namespace,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update,versions=v1,name=vnamespace-v1.kb.io,admissionReviewVersions=v1

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
func (v *NamespaceCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespace, ok := obj.(*k8siov1.Namespace)
	if !ok {
		return nil, fmt.Errorf("expected a Namespace object but got %T", obj)
	}
	namespacelog.Info("Validation for Namespace upon creation", "name", namespace.GetName())

	// Validation logic: Check if namespace has label "vault-injection" set to "enabled"
	if namespace.Labels == nil || namespace.Labels["vault-injection"] != "enabled" {
		return nil, fmt.Errorf("namespace %s must have label 'vault-injection=enabled' to be created", namespace.GetName())
	}

  ctx := context.Background()

	vaultClient, err = vault.New(
		vault.WithAddress(os.Getenv("VAULT_ADDR")),
		vault.WithRequestTimeout(30 * time.Second),
	)
	if err != nil {
		namespacelog.Error(err, "Failed to initialize Vault client")
		return nil, err
	}

	// read JWT token for ServiceAccount
  jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
  if err != nil {
		namespacelog.Error(err, "Failed to read ServiceAccount JWT token")
    return nil, err
  }

	resp, err := vaultClient.Auth.AppRoleLogin(
	  ctx,
	  schema.AppRoleLoginRequest{
		  RoleId:   os.Getenv("VAULT_ROLE_ID"),
		  SecretId: string(jwt),
	  },
	  vault.WithMountPath("my/approle/path"), // optional, defaults to "approle"
  )
  if err != nil {
		namespacelog.Error(err, "Failed to authenticate with Vault")
		return nil, err
  }
	if err := vaultClient.SetToken(resp.Auth.ClientToken); err != nil {
		namespacelog.Error(err, "Failed to set Vault token")
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

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
