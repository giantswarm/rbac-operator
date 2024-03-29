/*
Copyright 2023.

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

package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RoleBindingTemplateSpec defines the desired state of RoleBindingTemplate
type RoleBindingTemplateSpec struct {
	Template RoleBindingTemplateResource `json:"template"`
	Scopes   RoleBindingTemplateScopes   `json:"scopes"`
}

// RoleBindingTemplateStatus defines the observed state of RoleBindingTemplate
type RoleBindingTemplateStatus struct {
	// Namespaces contains a list of namespaces the RoleBinding is currently applied to
	Namespaces []string `json:"namespaces,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// RoleBindingTemplate is the Schema for the rolebindingtemplates API
type RoleBindingTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RoleBindingTemplateSpec   `json:"spec,omitempty"`
	Status RoleBindingTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RoleBindingTemplateList contains a list of RoleBindingTemplate
type RoleBindingTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RoleBindingTemplate `json:"items"`
}

// RoleBindingTemplateResource describes the data needed to create a rolebinding from a template.
type RoleBindingTemplateResource struct {
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`

	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef rbacv1.RoleRef `json:"roleRef"`
}

// RoleBindingTemplateScopes describes the scopes the RoleBindingTemplate should be applied to
type RoleBindingTemplateScopes struct {
	OrganizationSelector ScopeSelector `json:"organizationSelector"`
}

// ScopeSelector wraps a k8s label selector
type ScopeSelector struct {
	MatchLabels      map[string]string                 `json:"matchLabels,omitempty"`
	MatchExpressions []metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

func init() {
	SchemeBuilder.Register(&RoleBindingTemplate{}, &RoleBindingTemplateList{})
}
