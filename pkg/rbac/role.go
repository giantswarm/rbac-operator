package rbac

import (
    "fmt"
    "k8s.io/api/rbac/v1"
)

// Role represents a set of permissions
type Role struct {
    Rules []v1.PolicyRule `json:"rules" protobuf:"bytes,2,rep,name=rules"`
}

// NewRole creates a new Role with given rules
func NewRole(rules []v1.PolicyRule) *Role {
    return &Role{
        Rules: rules,
    }
}

// Validate ensures the role does not contain cluster-admin permissions
func (r *Role) Validate() error {
    for _, rule := range r.Rules {
        for _, verb := range rule.Verbs {
            if verb == "cluster-admin" {
                return fmt.Errorf("role cannot have cluster-admin permissions")
            }
        }
    }
    return nil
}
