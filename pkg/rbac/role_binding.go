package rbac

import (
    "fmt"
    "k8s.io/api/rbac/v1"
)

// RoleBinding maps a role to a user or set of users
type RoleBinding struct {
    RoleRef v1.RoleRef `json:"roleRef" protobuf:"bytes,2,opt,name=roleRef"`
}

// NewRoleBinding creates a new RoleBinding with given role reference
func NewRoleBinding(roleRef v1.RoleRef) *RoleBinding {
    return &RoleBinding{
        RoleRef: roleRef,
    }
}

// Validate ensures the role binding is not to cluster-admin
func (rb *RoleBinding) Validate() error {
    if rb.RoleRef.Name == "cluster-admin" {
        return fmt.Errorf("role binding cannot be to cluster-admin")
    }
    return nil
}
