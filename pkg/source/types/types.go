package types

import (
	"context"

	"cuelang.org/go/cue"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
)

const (
	TypeFieldName       = "type"
	PropertiesFieldName = "properties"
)

type Source interface {
	New() Source
	Init(properties cue.Value, filters []filtertypes.FilterMeta, filterRegistry *filterregistry.Registry) error
	AddEventHandler(eh EventHandler)
	Run(ctx context.Context) error
	Type() string
}

type SourceMeta struct {
	Type       string
	Properties cue.Value
}

type EventHandler func(sourceType string, event interface{})
