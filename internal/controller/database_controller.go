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
	"github.com/cloud-for-you/vault-secret-injector/internal/vault"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vaultsecret.cfy.cz,resources=databases/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=get;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Database object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	var database vaultsecretv1.Database
	if err := r.Get(ctx, req.NamespacedName, &database); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Reconciling Database", "name", database.Name, "namespace", database.Namespace)

	// For static creds, hardcoded path
	mount := "database"
	path := "static-creds/static"

	// Parse Annotations
	annotations, err := vaultsecretv1.ParseAnnotations(database.ObjectMeta)
	if err != nil {
		log.Error(err, "Failed to parse Database annotations", "name", database.Name, "namespace", database.Namespace)
		return ctrl.Result{}, err
	}

	// Validate configuration
	if annotations.VaultPath == "" && len(database.Spec.StringTemplate) == 0 {
		err := fmt.Errorf("neither vault path annotation nor stringTemplate specified")
		log.Error(err, "Invalid configuration", "name", database.Name, "namespace", database.Namespace)
		return ctrl.Result{}, err
	}

	// Setup Vault client
	vaultClient, impersonateJwt, err := vault.SetupVaultClient(ctx, database.ObjectMeta)
	if err != nil {
		database.Status.Message = "Failed to setup Vault client: " + err.Error()
		if updateErr := r.Status().Update(ctx, &database); updateErr != nil {
			log.Error(updateErr, "Failed to update Database status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Fetch database secret
	vaultDatabaseData, err := vault.FetchSecretEngineDatabase(vaultClient, impersonateJwt, mount, path)
	if err != nil {
		database.Status.Message = "Failed to fetch database secret: " + err.Error()
		if updateErr := r.Status().Update(ctx, &database); updateErr != nil {
			log.Error(updateErr, "Failed to update Database status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Allowed keys in vaultDatabaseData:
	// last_vault_rotation, Type: string
	// password, Type: string
	// rotation_period, Type: json.Number
	// ttl, Type: json.Number
	// username, Type: string
	secretData := make(map[string][]byte)
	for key, tmpl := range database.Spec.StringTemplate {
		templatedString, err := vaultsecretv1.TemplatingStringData(tmpl, vaultDatabaseData.Data)
		if err != nil {
			database.Status.Message = "Failed to prepare secret data for key " + key + ": " + err.Error()
			if updateErr := r.Status().Update(ctx, &database); updateErr != nil {
				log.Error(updateErr, "Failed to update Database status")
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		}
		secretData[key] = []byte(templatedString)
	}

	// Check if Kubernetes Secret already exists
	secretExists := true
	k8sSecretCheck := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Name: database.Name, Namespace: database.Namespace}, k8sSecretCheck)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		secretExists = false
	}

	// Handle secret creation/update and status
	changed, err := database.HandleSecretAndStatus(ctx, r.Client, r.Status(), secretData)
	if err != nil {
		database.Status.Message = "Failed to handle secret and status: " + err.Error()
		if updateErr := r.Status().Update(ctx, &database); updateErr != nil {
			log.Error(updateErr, "Failed to update Database status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Trigger rollouts
	err = r.TriggerRollouts(ctx, &database, changed, secretExists)
	if err != nil {
		log.Error(err, "Failed to trigger rollouts")
		database.Status.Message = "Failed to trigger rollouts: " + err.Error()
		if updateErr := r.Status().Update(ctx, &database); updateErr != nil {
			log.Error(updateErr, "Failed to update Database status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled Database", "name", database.Name, "namespace", database.Namespace)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultsecretv1.Database{}).
		Named("database").
		Complete(r)
}
