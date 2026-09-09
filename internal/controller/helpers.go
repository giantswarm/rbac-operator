package controller

import (
	"cmp"
	"fmt"
	"slices"

	rbacv1 "k8s.io/api/rbac/v1"
)

// uniqueSubjectsFromRBs deduplicates subjects across all RoleBindings.
func uniqueSubjectsFromRBs(rbs []rbacv1.RoleBinding) []rbacv1.Subject {
	seen := map[string]struct{}{}
	var out []rbacv1.Subject
	for _, rb := range rbs {
		for _, s := range rb.Subjects {
			k := fmt.Sprintf("%s/%s/%s", s.Kind, s.Namespace, s.Name)
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				out = append(out, s)
			}
		}
	}
	sortSubjects(out)
	return out
}

// subjectsFromRBsWithClusterRoleRef returns unique subjects from RoleBindings referencing a specific ClusterRole.
func subjectsFromRBsWithClusterRoleRef(rbs []rbacv1.RoleBinding, clusterRoleName string) []rbacv1.Subject {
	seen := map[string]struct{}{}
	var out []rbacv1.Subject
	for _, rb := range rbs {
		if rb.RoleRef.Kind == "ClusterRole" && rb.RoleRef.Name == clusterRoleName {
			for _, s := range rb.Subjects {
				k := fmt.Sprintf("%s/%s/%s", s.Kind, s.Namespace, s.Name)
				if _, ok := seen[k]; !ok {
					seen[k] = struct{}{}
					out = append(out, s)
				}
			}
		}
	}
	sortSubjects(out)
	return out
}

// sortSubjects orders subjects deterministically. Cache lists come back in
// random order, so unsorted subjects would make CreateOrUpdate see a change
// on every reconcile and rewrite the binding.
func sortSubjects(subjects []rbacv1.Subject) {
	slices.SortFunc(subjects, func(a, b rbacv1.Subject) int {
		return cmp.Or(
			cmp.Compare(a.Kind, b.Kind),
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Name, b.Name),
		)
	})
}
