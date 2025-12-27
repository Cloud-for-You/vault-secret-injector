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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cfyczv1 "github.com/cloud-for-you/vault-secret-injector/api/v1"
)

var _ = Describe("VaultSecret Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		vaultsecret := &cfyczv1.VaultSecret{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VaultSecret")
			err := k8sClient.Get(ctx, typeNamespacedName, vaultsecret)
			if err != nil && errors.IsNotFound(err) {
				resource := &cfyczv1.VaultSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Annotations: map[string]string{
							"vault.hashicorp.com/path": "test/path",
						},
					},
					Spec: cfyczv1.VaultSecretSpec{
						StringData: map[string]cfyczv1.VaultSecretDataValue{
							"key1": cfyczv1.VaultSecretDataValue("test/path@field1"),
							"key2": cfyczv1.VaultSecretDataValue("test/path@field2"),
						},
						Type:      "Opaque",
						Immutable: false,
						RolloutObjectRef: []cfyczv1.RolloutObjectRef{
							{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "test-deployment",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &cfyczv1.VaultSecret{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VaultSecret")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VaultSecretReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// In test environment, vault is not set up, so expect error
			Expect(err).To(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
