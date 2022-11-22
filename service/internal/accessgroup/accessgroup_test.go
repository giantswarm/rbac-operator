package accessgroup

import "testing"

func Test_LegacyGroupsAreApplied(t *testing.T) {
	accessGroups := []AccessGroup{
		{Name: "group1"},
		{Name: "group2"},
	}

	accessGroups = addLegacyGroupIfMissing(accessGroups, "legacy:group1")
	if len(accessGroups) != 3 {
		t.Fatalf("Incorrect length of access groups - expected: 3, actual: %d", len(accessGroups))
	}

	accessGroups = addLegacyGroupIfMissing(accessGroups, "group1")
	if len(accessGroups) != 3 {
		t.Fatalf("Incorrect length of access groups - expected: 3, actual: %d", len(accessGroups))
	}
}
