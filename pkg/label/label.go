package label

const (
	// LegacyCustomer Labels, used in legacy cluster namespaces
	LegacyCustomer = "customer"
)

type LabelsGetter interface {
	GetLabels() map[string]string
}
