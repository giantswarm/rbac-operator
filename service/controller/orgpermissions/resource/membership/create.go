package membership

import (
	"context"
)

// Ensures that a ClusterRoleBinding '<rolebinding-name>-organization-<organization>-read'
// exists for a certain RoleBinding, with the same subjects.
//
// DEPRECATED
// This resource is replaced by the rbac/externalresources resource.
// The deletion logic remains to help cleaning up
//
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	return r.EnsureDeleted(ctx, obj)
}
