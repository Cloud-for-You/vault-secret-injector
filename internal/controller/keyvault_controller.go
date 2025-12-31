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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	vaultsecretv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
	vaultlib "github.com/cloud-for-you/vault-secret-injector/internal/vault"
)

// KeyVaultReconciler reconciles a KeyVault object
type KeyVaultReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=keyvaults,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=keyvaults/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=keyvaults/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=get;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VaultSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *KeyVaultReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	var keyVault vaultsecretv1.KeyVault
	if err := r.Get(ctx, req.NamespacedName, &keyVault); err != nil {
		// handle error
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Reconciling KeyVault", "name", keyVault.Name, "namespace", keyVault.Namespace)

	// Parse Annotations
	annotations, err := vaultsecretv1.ParseAnnotations(&keyVault.ObjectMeta)
	if err != nil {
		log.Error(err, "Failed to parse KeyVault annotations", "name", keyVault.Name, "namespace", keyVault.Namespace)
		return ctrl.Result{}, err
	}

	// Validate configuration
	if annotations.VaultPath == "" && len(keyVault.Spec.StringData) == 0 {
		err := fmt.Errorf("neither vault path annotation nor stringData specified")
		log.Error(err, "Invalid configuration", "name", keyVault.Name, "namespace", keyVault.Namespace)
		return ctrl.Result{}, err
	}

	// Setup Vault client
	vaultClient, impersonateJwt, err := vaultlib.SetupVaultClient(ctx, keyVault.ObjectMeta)
	if err != nil {
		keyVault.Status.Message = "Failed to setup Vault client: " + err.Error()
		if updateErr := r.Status().Update(ctx, &keyVault); updateErr != nil {
			log.Error(updateErr, "Failed to update KeyVault status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Fetch secret data
	secretData, err := vaultlib.FetchKVSecret(vaultClient, impersonateJwt, &annotations, &keyVault)
	if err != nil {
		keyVault.Status.Message = "Failed to fetch secret data: " + err.Error()
		if updateErr := r.Status().Update(ctx, &keyVault); updateErr != nil {
			log.Error(updateErr, "Failed to update KeyVault status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Check if Kubernetes Secret already exists
	secretExists := true
	k8sSecretCheck := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Name: annotations.VaultSecretName, Namespace: keyVault.Namespace}, k8sSecretCheck)
	if err != nil {
		if client.IgnoreNotFound(err) != nil { // Ignore NotFound, but return other errors
			return ctrl.Result{}, err
		}
		secretExists = false // Secret does not exist, so this will be a create operation
	}

	// Handle secret creation/update and status
	changed, err := keyVault.HandleSecretAndStatus(ctx, r.Client, r.Status(), secretData)
	if err != nil {
		keyVault.Status.Message = "Failed to handle secret and status: " + err.Error()
		if updateErr := r.Status().Update(ctx, &keyVault); updateErr != nil {
			log.Error(updateErr, "Failed to update KeyVault status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Trigger rollouts
	err = r.TriggerRollouts(ctx, &keyVault, changed, secretExists)
	if err != nil {
		log.Error(err, "Failed to trigger rollouts")
		keyVault.Status.Message = "Failed to trigger rollouts: " + err.Error()
		if updateErr := r.Status().Update(ctx, &keyVault); updateErr != nil {
			log.Error(updateErr, "Failed to update KeyVault status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled VaultSecret", "name", keyVault.Name, "namespace", keyVault.Namespace)

	if annotations.VaultRefreshInterval > 0 {
		// Requeue after the specified refresh interval
		return ctrl.Result{RequeueAfter: annotations.VaultRefreshInterval}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeyVaultReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultsecretv1.KeyVault{}).
		Named("keyvault").
		Complete(r)
}
