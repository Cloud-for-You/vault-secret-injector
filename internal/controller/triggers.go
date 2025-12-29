package controller

import (
	"context"

	vaultsecretv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// triggerRollouts triggers rollouts for the specified objects if secret changed and secret existed.
func (r *KeyVaultReconciler) TriggerRollouts(ctx context.Context, vaultSecret *vaultsecretv1.KeyVault, changed bool, secretExists bool) error {
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

// triggerRollouts triggers rollouts for the specified objects if secret changed and secret existed.
func (r *DatabaseReconciler) TriggerRollouts(ctx context.Context, database *vaultsecretv1.Database, changed bool, secretExists bool) error {
	if !changed || !secretExists {
		return nil
	}
	for _, rolloutRef := range database.Spec.RolloutObjectRef {
		err := rolloutRef.TriggerRollout(ctx, r.Client, database.GetNamespace())
		if err != nil {
			return err
		}
		logf.FromContext(ctx).Info("Triggered rollout for object", "apiVersion", rolloutRef.APIVersion, "kind", rolloutRef.Kind, "name", rolloutRef.Name)
	}
	return nil
}
