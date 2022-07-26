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

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Action is an interface for actions. Anything that implements this interface
// can be registered as Actions, and will be executed automatically.
type Action interface {
	// New returns a new uninitialized instance.
	New() Action

	// Validate validates properties.
	Validate(properties map[string]interface{}) error

	// Init initializes this instance using user-provided properties and common things.
	// Typically, this will only be called once and the initialized instance
	// will be cached. Subsequent calls will use the Run() method in cached instances.
	Init(c Common, properties map[string]interface{}) error

	// Run executes this Action.
	//
	// sourceType is what type the Source is.
	//
	// event is what event happened, containing a brief event object. Do not include
	// complex objects in it. For example, a k8s-resource-watcher Source may contain
	// what event happened (create, update, delete) in it.
	//
	// data is the detailed event, for machines to process, e.g. passed to filters to
	// do filtering . You may put complex objects in it. For example,
	// a k8s-resource-watcher Source may contain the entire object that is changed
	// in it.
	//
	// messages are the messages that the filters left to you.
	//
	// Run will be called automatically job workers. Since this method
	// will be called multiple times, you should not store any states in your Action.
	Run(ctx context.Context, sourceType string, event interface{}, data interface{}, messages []string) error

	// Type returns the type of this Action. Since this is an Action, please name
	// your action as do-something, instead of something-doer.
	Type() string

	// AllowConcurrency indicates if this Action can be executed concurrently.
	// If not, only one instance of this Action type can run at a time.
	AllowConcurrency() bool
}

// Common is some common things that are passed to Actions when they initialize.
type Common struct {
	Client client.Client
}

// ActionMeta is what users type in their configurations, specifying what action
// they want to use and what properties they provided.
type ActionMeta struct {
	// Type is the name (identifier) of this action.
	Type string `json:"type"`

	// Properties are user-provided parameters. You should parse it yourself.
	Properties map[string]interface{} `json:"properties"`

	// Raw is the raw string representation of this action. Typically, you will
	// not use it. This is for identifying action instances.
	Raw string `json:"raw,omitempty"`
}
