package accessgroup

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

func (a *AccessGroups) Validate() bool {
	return len(a.WriteAllCustomerGroups) > 0 && len(a.WriteAllGiantswarmGroups) > 0
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
