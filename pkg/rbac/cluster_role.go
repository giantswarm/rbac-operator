package rbac

import (
    "fmt"
    "k8s.io/api/rbac/v1"
)

// ClusterRole represents cluster-wide permissions
type ClusterRole struct {
    Rules []v1.PolicyRule `json:"rules" protobuf:"bytes,2,rep,name=rules"`
}

// NewClusterRole creates a new ClusterRole with given rules
func NewClusterRole(rules []v1.PolicyRule) *ClusterRole {
    return &ClusterRole{
        Rules: rules,
    }
}

// Validate ensures the cluster role does not contain cluster-admin permissions
func (cr *ClusterRole) Validate() error {
    for _, rule := range cr.Rules {
        for _, verb := range rule.Verbs {
            if verb == "cluster-admin" {
                return fmt.Errorf("cluster role cannot have cluster-admin permissions")
            }
        }
    }
    return nil
}
