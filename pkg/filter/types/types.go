package types

import (
	"cuelang.org/go/cue"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TypeFieldName       = "type"
	PropertiesFieldName = "properties"
)

type Filter interface {
	// New returns a new uninitialized instance.
	New() Filter
	// Init initializes this instance using user-provided properties.
	Init(properties cue.Value) error
	// ApplyToObject applies this Filter to the given object that came from
	// sources. Returning false will filter this object out.
	ApplyToObject(obj metav1.Object) (bool, error)
	// Type returns the type of this Filter.
	Type() string
}

type FilterMeta struct {
	Type       string
	Properties cue.Value
	Raw        string
}
