package label

const (
	Cluster = "giantswarm.io/cluster"

	Organization = "giantswarm.io/organization"

	ManagedBy = "giantswarm.io/managed-by"

	// LegacyCustomer Labels, used in legacy cluster namespaces
	LegacyCustomer = "customer"

	NOTES = "giantswarm.io/notes"
)

type LabelsGetter interface {
	GetLabels() map[string]string
}
