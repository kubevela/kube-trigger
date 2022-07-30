package types

import (
	"context"

	"cuelang.org/go/cue"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TypeFieldName       = "type"
	PropertiesFieldName = "properties"
)

type Action interface {
	// New returns a new uninitialized instance.
	New() Action
	// Init initializes this instance using user-provided properties.
	Init(c Common, properties cue.Value) error
	// Run executes this Action.
	Run(ctx context.Context, sourceType string, event interface{}) error
	// Type returns the type of this Action.
	Type() string
	AllowConcurrent() bool
}

type Common struct {
	Client client.Client
}

type ActionMeta struct {
	Type       string
	Properties cue.Value
	Raw        string
}
