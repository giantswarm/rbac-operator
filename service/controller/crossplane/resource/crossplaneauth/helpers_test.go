package crossplaneauth_test

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/rbac-operator/service/controller/crossplane/key"
)

var crossplaneEditCR = rbacv1.ClusterRole{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ClusterRole",
		APIVersion: "rbac.authorization.k8s.io",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "crossplane-edit",
	},
}

var crossplaneClusterRoleBinding = rbacv1.ClusterRoleBinding{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: key.GetClusterRoleBindingName("crossplane-edit"),
	},
}
