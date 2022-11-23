package accessgroup

import rbacv1 "k8s.io/api/rbac/v1"

type AccessGroup struct {
	Name string
}

type AccessGroups struct {
	WriteAllCustomerGroups   []AccessGroup
	WriteAllGiantswarmGroups []AccessGroup
}

func (a *AccessGroups) AddLegacyCustomerAdminGroup(legacyGroupName string) {
	a.WriteAllCustomerGroups = addLegacyGroupIfMissing(a.WriteAllCustomerGroups, legacyGroupName)
}

func (a *AccessGroups) AddLegacyGiantswarmAdminGroup(legacyGroupName string) {
	a.WriteAllGiantswarmGroups = addLegacyGroupIfMissing(a.WriteAllGiantswarmGroups, legacyGroupName)
}

func (a *AccessGroups) HasValidWriteAllGiantswarmAdminGroups() bool {
	return ValidateGroups(a.WriteAllGiantswarmGroups)
}

func addLegacyGroupIfMissing(groups []AccessGroup, legacyGroupName string) []AccessGroup {
	if legacyGroupName == "" {
		return groups
	}
	for _, group := range groups {
		if group.Name == legacyGroupName {
			return groups
		}
	}
	return append(groups, AccessGroup{Name: legacyGroupName})
}

func GroupsToSubjects(groups []AccessGroup) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, group := range groups {
		if group.Name != "" {
			subjects = append(subjects, rbacv1.Subject{
				Kind: "Group",
				Name: group.Name,
			})
		}
	}
	return subjects
}

func ValidateGroups(groups []AccessGroup) bool {
	for _, group := range groups {
		if group.Name != "" {
			return true
		}
	}
	return false
}
