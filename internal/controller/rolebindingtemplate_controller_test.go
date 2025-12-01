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
	"time"

	"github.com/giantswarm/k8smetadata/pkg/label"
	security "github.com/giantswarm/organization-operator/api/v1alpha1"
	. "github.com/giantswarm/rbac-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RoleBindingTemplate Controller", func() {
	const (
		resourceName = "test-resource"

		timeout  = time.Second * 5
		interval = time.Millisecond * 250
	)
	organizations := [...]string{"example-1", "example-2"}
	expectedNamespaces := []string{
		getOrgNsName("example-1"),
		getOrgNsName("example-2"),
	}

	ctx := context.Background()

	typeNamespacedName := types.NamespacedName{
		Name: resourceName,
	}
	rolebindingtemplate := &RoleBindingTemplate{}

	BeforeEach(func() {
		By("creating organizations")
		for _, org := range organizations {
			Expect(k8sClient.Create(ctx, getTestOrganization(org))).To(Succeed())
		}

		// NOTE: Can't delete namespaces in envtest
		// https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
		By("creating namespaces for organizations")
		for _, org := range organizations {
			// NOTE: Namespaces will exist after first creation as envtest does not support namespace deletion
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, getTestOrgNamespace(org)))).To(Succeed())
		}

		DeferCleanup(func() {
			By("deleting organizations")
			for _, org := range organizations {
				Expect(k8sClient.Delete(ctx, getTestOrganization(org))).To(Succeed())
			}
		})
	})

	When("creating a RoleBindingTemplate", func() {
		BeforeEach(func() {
			rolebindingtemplate := &RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: RoleBindingTemplateSpec{
					Template: RoleBindingTemplateResource{
						RoleRef: rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     "example-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "ServiceAccount",
								Name: "example-sa",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rolebindingtemplate)).To(Succeed())

			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, rolebindingtemplate)).To(Succeed())

				// NOTE: No garbage collection in envtest
				// https://book.kubebuilder.io/reference/envtest.html#testing-considerations
				By("cleaning up all rolebindings")
				rolebindings := &rbacv1.RoleBindingList{}
				Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
				for _, rb := range rolebindings.Items {
					Expect(k8sClient.Delete(ctx, &rb)).To(Succeed())
				}
			})
		})

		It("should update status correctly", func() {
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, rolebindingtemplate)).To(Succeed())
				g.Expect(apimeta.IsStatusConditionTrue(rolebindingtemplate.Status.Conditions, ReadyCondition)).To(BeTrue())
				g.Expect(rolebindingtemplate.Status.ProvisionedNamespaces).To(Equal(expectedNamespaces))
				g.Expect(rolebindingtemplate.Status.FailedNamespaces).To(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		// NOTE: No garbage collection in envtest, just verifying owner references
		// https://book.kubebuilder.io/reference/envtest.html#testing-considerations
		It("should set owner references correctly", func() {
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, rolebindingtemplate)).To(Succeed())
				g.Expect(apimeta.IsStatusConditionTrue(rolebindingtemplate.Status.Conditions, ReadyCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			expectedOwnerRef := metav1.OwnerReference{
				Kind:               "RoleBindingTemplate",
				APIVersion:         GroupVersion.String(),
				Name:               rolebindingtemplate.Name,
				UID:                rolebindingtemplate.UID,
				Controller:         ptr.To(true),
				BlockOwnerDeletion: ptr.To(true),
			}

			rolebindings := &rbacv1.RoleBindingList{}
			Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
			for _, ns := range expectedNamespaces {
				var rb *rbacv1.RoleBinding
				for i := range rolebindings.Items {
					if rolebindings.Items[i].Namespace == ns {
						rb = &rolebindings.Items[i]
						break
					}
				}
				Expect(rb.OwnerReferences).To(ContainElement(expectedOwnerRef))
			}
		})

		It("should create correct rolebindings in all organization namespaces", func() {
			By("waiting for the RoleBindingTemplate to be ready")
			rolebindingtemplate := &RoleBindingTemplate{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, rolebindingtemplate)).To(Succeed())
				g.Expect(apimeta.IsStatusConditionTrue(rolebindingtemplate.Status.Conditions, ReadyCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			By("verifying generated RoleBindings match the template")
			rolebindings := &rbacv1.RoleBindingList{}
			Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
			for _, ns := range expectedNamespaces {
				var rb *rbacv1.RoleBinding
				for i := range rolebindings.Items {
					if rolebindings.Items[i].Namespace == ns {
						rb = &rolebindings.Items[i]
						break
					}
				}
				Expect(rb).NotTo(BeNil())
				Expect(rb.Name).To(Equal(resourceName))
				Expect(rb.RoleRef).To(Equal(rolebindingtemplate.Spec.Template.RoleRef))
				Expect(rb.Subjects).To(Equal(rolebindingtemplate.Spec.Template.Subjects))
			}
		})
	})

	When("creating a RoleBindingTemplate with organization selectors", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, rolebindingtemplate)).To(Succeed())

			// NOTE: No garbage collection in envtest
			// https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("cleaning up all rolebindings")
			rolebindings := &rbacv1.RoleBindingList{}
			Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
			for _, rb := range rolebindings.Items {
				Expect(k8sClient.Delete(ctx, &rb)).To(Succeed())
			}
		})

		It("should create correct rolebindings in one namespaces", func() {
			rolebindingtemplate = &RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: RoleBindingTemplateSpec{
					Template: RoleBindingTemplateResource{
						RoleRef: rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     "example-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "ServiceAccount",
								Name: "example-sa",
							},
						},
					},
					Scopes: RoleBindingTemplateScopes{
						OrganizationSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"example-key": organizations[0],
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rolebindingtemplate)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, rolebindingtemplate)).To(Succeed())
				g.Expect(apimeta.IsStatusConditionTrue(rolebindingtemplate.Status.Conditions, ReadyCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			rolebindings := &rbacv1.RoleBindingList{}
			Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
			Expect(rolebindings.Items).To(HaveLen(1))
		})

		It("should not create any RoleBindings", func() {
			rolebindingtemplate = &RoleBindingTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: RoleBindingTemplateSpec{
					Template: RoleBindingTemplateResource{
						RoleRef: rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     "example-role",
						},
						Subjects: []rbacv1.Subject{
							{
								Kind: "ServiceAccount",
								Name: "example-sa",
							},
						},
					},
					Scopes: RoleBindingTemplateScopes{
						OrganizationSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"non-existent-key": "non-existent-value",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, rolebindingtemplate)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, rolebindingtemplate)).To(Succeed())
				g.Expect(apimeta.IsStatusConditionTrue(rolebindingtemplate.Status.Conditions, ReadyCondition)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			rolebindings := &rbacv1.RoleBindingList{}
			Expect(k8sClient.List(ctx, rolebindings)).To(Succeed())
			Expect(rolebindings.Items).To(BeEmpty())
		})
	})
})

func getTestOrganization(name string) *security.Organization {
	return &security.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				name:          "example-value",
				"example-key": name,
			},
		},
		Spec: security.OrganizationSpec{},
	}
}

func getTestOrgNamespace(orgName string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: getOrgNsName(orgName),
			Labels: map[string]string{
				label.Organization: orgName,
				label.Cluster:      fmt.Sprintf("cluster-%s", orgName),
			},
		},
	}
}

func getOrgNsName(orgName string) string {
	return fmt.Sprintf("org-%s", orgName)
}
