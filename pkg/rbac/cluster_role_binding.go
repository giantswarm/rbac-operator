package rbac

import (
    "fmt"
    "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding maps a cluster role to a user or set of users
type ClusterRoleBinding struct {
    RoleRef v1.RoleRef `json:"roleRef" protobuf:"bytes,2,opt,name=roleRef"`
}

// NewClusterRoleBinding creates a new ClusterRoleBinding with given role reference
func NewClusterRoleBinding(roleRef v1.RoleRef) *ClusterRoleBinding {
    return &ClusterRoleBinding{
        RoleRef: roleRef,
    }
}

// Validate ensures the cluster role binding is not to cluster-admin
func (crb *ClusterRoleBinding) Validate() error {
    if crb.RoleRef.Name == "cluster-admin" {
        return fmt.Errorf("cluster role binding cannot be to cluster-admin")
    }
    return nil
}
