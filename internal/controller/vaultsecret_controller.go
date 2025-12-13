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

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cfyczv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
	vaultlib "github.com/cloud-for-you/vault-secret-injector/internal/vault"
)

// VaultSecretReconciler reconciles a VaultSecret object
type VaultSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cfy.cz,resources=vaultsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cfy.cz,resources=vaultsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cfy.cz,resources=vaultsecrets/finalizers,verbs=update
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
func (r *VaultSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here
	var vaultSecret cfyczv1.VaultSecret
	if err := r.Get(ctx, req.NamespacedName, &vaultSecret); err != nil {
		// handle error
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Log.Info("Reconciling VaultSecret", "name", vaultSecret.Name, "namespace", vaultSecret.Namespace)

	// Get Impersonate Service Account Token
	clientset := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	impersonateJwt, err := getImpersonateSAToken(ctx, clientset, vaultSecret.GetNamespace(), "default", "serviceaccount", int64(600))
	if err != nil {
		log.Log.Error(err, "Failed to get impersonated service account token")
		return ctrl.Result{}, err
	}
	log.Log.Info("Successfully obtained impersonated service account token")
	
	// Create Vault Client
  vaultClient, err := vaultlib.NewVaultClient()
  if err != nil {
    log.Log.Error(err, "Failed to create Vault client")
    return ctrl.Result{}, err
  }

	// Login to Vault with K8s Auth Method
	err = vaultlib.VaultLoginWithK8sAuth(ctx, vaultClient, "k8s-kind", impersonateJwt, vaultSecret.GetNamespace())
	if err != nil {
		log.Log.Error(err, "Failed to login to Vault")
		return ctrl.Result{}, err
	}
  log.Log.Info("Successfully logged in to Vault")

	// Fetch data from Vault KV engine
	kvPath := "test"
	secretData, err := vaultlib.FetchSecretEngineKV(vaultClient, impersonateJwt, kvPath)
	if err != nil {
		log.Log.Error(err, "Failed to fetch secret from Vault", "path", kvPath)
		return ctrl.Result{}, err
	}
	log.Log.Info("Successfully fetched secret from Vault", "path", kvPath)

	// Print fetched data in JSON format (only for demonstration; avoid in production)
	log.Log.Info("Fetched secret data", "data", secretData)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cfyczv1.VaultSecret{}).
		Named("vaultsecret").
		Complete(r)
}

// GetImpersonateSAToken requests a service account token.
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
