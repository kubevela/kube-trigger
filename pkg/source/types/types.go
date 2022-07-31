/*
Copyright 2022 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"context"

	"cuelang.org/go/cue"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	"github.com/kubevela/kube-trigger/pkg/source/eventhandler"
)

const (
	TypeFieldName       = "type"
	PropertiesFieldName = "properties"
)

// Source is an interface for sources. Anything that implements this interface
// can be registered as Sources, and will be executed automatically.
type Source interface {
	// New returns a new uninitialized instance.
	New() Source

	// Init initializes this instance using user-provided properties and filters.
	// These filters are attached to this source. You should filter events
	// using these provided filters.
	Init(properties cue.Value, filters []filtertypes.FilterMeta, filterRegistry *filterregistry.Registry) error

	// AddEventHandlers will be called to give you all the EventHandlers that
	// you should call when an event happened (after filtering).
	AddEventHandlers(ehs eventhandler.Store)

	// Run starts this Source. You should handle the context so that you can
	// know when to exit.
	Run(ctx context.Context) error

	// Type returns the type of this Source.
	Type() string
}

// SourceMeta is what users type in their configurations, specifying what source
// they want to use and what properties they provided.
type SourceMeta struct {
	// Type is the name (identifier) of this Source.
	Type string

	// Properties are user-provided parameters. You should parse it yourself.
	Properties cue.Value
}
