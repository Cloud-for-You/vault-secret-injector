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
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=get;create

// setupVaultClient sets up the Vault client and authenticates.
func (r *VaultSecretReconciler) setupVaultClient(ctx context.Context, vaultSecret *cfyczv1.VaultSecret) (*vaultapi.Client, string, error) {
	clientset := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	impersonateJwt, err := getImpersonateSAToken(ctx, clientset, vaultSecret.GetNamespace(), "default", "serviceaccount", int64(600))
	if err != nil {
		return nil, "", err
	}
	vaultlib.LogAudit(impersonateJwt, "Obtained impersonated service account token", map[string]interface{}{"namespace": vaultSecret.GetNamespace(), "serviceAccount": "default"})

	vaultClient, err := vaultlib.NewVaultClient()
	if err != nil {
		return nil, "", err
	}

	err = vaultlib.VaultLoginWithK8sAuth(ctx, vaultClient, "k8s-kind", impersonateJwt, vaultSecret.GetNamespace())
	if err != nil {
		return nil, "", err
	}
	return vaultClient, impersonateJwt, nil
}

// fetchSecretData fetches secret data from Vault based on annotations and spec.
func (r *VaultSecretReconciler) fetchSecretData(vaultClient *vaultapi.Client, impersonateJwt string, annotations *cfyczv1.VaultSecretAnnotations, vaultSecret *cfyczv1.VaultSecret) (map[string][]byte, error) {
	var secretData map[string][]byte
	if annotations.VaultPath != "" {
		data, err := vaultlib.FetchSecretEngineKV(vaultClient, impersonateJwt, annotations.VaultMount, annotations.VaultPath)
		if err != nil {
			return nil, err
		}
		secretData = data
	} else {
		secretData = make(map[string][]byte)
		for secretKey, vaultSpec := range vaultSecret.Spec.StringData {
			parts := strings.Split(vaultSpec, "@")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid stringData format for key %s: expected <vaultPath>@<key>", secretKey)
			}
			vaultPath := parts[0]
			keyInVault := parts[1]
			data, err := vaultlib.FetchSecretEngineKV(vaultClient, impersonateJwt, annotations.VaultMount, vaultPath)
			if err != nil {
				return nil, err
			}
			value, ok := data[keyInVault]
			if !ok {
				return nil, fmt.Errorf("key %s not found in vault path %s", keyInVault, vaultPath)
			}
			secretData[secretKey] = value
		}
	}
	return secretData, nil
}

// handleSecretAndStatus creates or updates the K8s secret and updates the VaultSecret status.
func (r *VaultSecretReconciler) handleSecretAndStatus(ctx context.Context, vaultSecret *cfyczv1.VaultSecret, secretData map[string][]byte) (bool, error) {
	changed, err := vaultSecret.CreateOrUpdateK8sSecret(ctx, r.Client, secretData)
	if err != nil {
		return false, err
	}

	// Update LastUpdated timestamp only if data changed
	if changed {
		vaultSecret.Status.LastUpdated = metav1.Now().Format(time.RFC3339)
	}
	if updateErr := r.Status().Update(ctx, vaultSecret); updateErr != nil {
		return false, updateErr
	}
	return changed, nil
}

// triggerRollouts triggers rollouts for the specified objects if secret changed and secret existed.
func (r *VaultSecretReconciler) triggerRollouts(ctx context.Context, vaultSecret *cfyczv1.VaultSecret, changed bool, secretExists bool) error {
	if !changed || !secretExists {
		return nil
	}
	for _, rolloutRef := range vaultSecret.Spec.RolloutObjectRef {
		err := rolloutRef.TriggerRollout(ctx, r.Client, vaultSecret.GetNamespace())
		if err != nil {
			return err
		}
		logf.FromContext(ctx).Info("Triggered rollout for object", "apiVersion", rolloutRef.APIVersion, "kind", rolloutRef.Kind, "name", rolloutRef.Name)
	}
	return nil
}

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
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	var vaultSecret cfyczv1.VaultSecret
	if err := r.Get(ctx, req.NamespacedName, &vaultSecret); err != nil {
		// handle error
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Reconciling VaultSecret", "name", vaultSecret.Name, "namespace", vaultSecret.Namespace)

	// Parse Annotations
	annotations, err := vaultSecret.ParseAnnotations(vaultSecret.ObjectMeta)
	if err != nil {
		log.Error(err, "Failed to parse VaultSecret annotations", "name", vaultSecret.Name, "namespace", vaultSecret.Namespace)
		return ctrl.Result{}, err
	}
	log.Info(annotations.VaultPath)

	// Validate configuration
	if annotations.VaultPath == "" && len(vaultSecret.Spec.StringData) == 0 {
		err := fmt.Errorf("neither vault path annotation nor stringData specified")
		log.Error(err, "Invalid configuration", "name", vaultSecret.Name, "namespace", vaultSecret.Namespace)
		return ctrl.Result{}, err
	}

	// Setup Vault client
	vaultClient, impersonateJwt, err := r.setupVaultClient(ctx, &vaultSecret)
	if err != nil {
		vaultSecret.Status.Message = "Failed to setup Vault client: " + err.Error()
		if updateErr := r.Status().Update(ctx, &vaultSecret); updateErr != nil {
			log.Error(updateErr, "Failed to update VaultSecret status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Fetch secret data
	secretData, err := r.fetchSecretData(vaultClient, impersonateJwt, &annotations, &vaultSecret)
	if err != nil {
		vaultSecret.Status.Message = "Failed to fetch secret data: " + err.Error()
		if updateErr := r.Status().Update(ctx, &vaultSecret); updateErr != nil {
			log.Error(updateErr, "Failed to update VaultSecret status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Check if Kubernetes Secret already exists
	secretExists := true
	k8sSecretCheck := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Name: annotations.VaultSecretName, Namespace: vaultSecret.Namespace}, k8sSecretCheck)
	if err != nil {
		if client.IgnoreNotFound(err) != nil { // Ignore NotFound, but return other errors
			return ctrl.Result{}, err
		}
		secretExists = false // Secret does not exist, so this will be a create operation
	}

	// Handle secret creation/update and status
	changed, err := r.handleSecretAndStatus(ctx, &vaultSecret, secretData)
	if err != nil {
		vaultSecret.Status.Message = "Failed to handle secret and status: " + err.Error()
		if updateErr := r.Status().Update(ctx, &vaultSecret); updateErr != nil {
			log.Error(updateErr, "Failed to update VaultSecret status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Trigger rollouts
	err = r.triggerRollouts(ctx, &vaultSecret, changed, secretExists)
	if err != nil {
		log.Error(err, "Failed to trigger rollouts")
		vaultSecret.Status.Message = "Failed to trigger rollouts: " + err.Error()
		if updateErr := r.Status().Update(ctx, &vaultSecret); updateErr != nil {
			log.Error(updateErr, "Failed to update VaultSecret status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled VaultSecret", "name", vaultSecret.Name, "namespace", vaultSecret.Namespace)

	if annotations.VaultRefreshInterval > 0 {
		// Requeue after the specified refresh interval
		return ctrl.Result{RequeueAfter: annotations.VaultRefreshInterval}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cfyczv1.VaultSecret{}).
		Named("vaultsecret").
		Complete(r)
}

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
