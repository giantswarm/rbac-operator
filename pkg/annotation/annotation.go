package annotation

const (
	// LegacyOrganization Annotation, used on organizations that were migrated previously
	LegacyOrganization = "ui.giantswarm.io/original-organization-name"
)

type AnnotationsGetter interface {
	GetAnnotations() map[string]string
}
