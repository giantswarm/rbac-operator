package label

const (
	Cluster = "giantswarm.io/cluster"

	Organization = "giantswarm.io/organization"

	ManagedBy = "giantswarm.io/managed-by"

	// Labels, used in legacy cluster namespaces
	LegacyCustomer = "customer"
)

type LabelsGetter interface {
	GetLabels() map[string]string
}
