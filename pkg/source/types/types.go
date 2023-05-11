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

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
)

// Source is an interface for sources. Anything that implements this interface
// can be registered as Sources, and will be executed automatically.
type Source interface {
	// New returns a new uninitialized instance.
	New() Source

	// Init initializes this instance using user-provided properties.
	// Call the EventHandler when an event happened.
	Init(properties *runtime.RawExtension, eh eventhandler.EventHandler) error

	// Run starts this Source. You should handle the context so that you can
	// know when to exit.
	Run(ctx context.Context) error

	// Type returns the type of this Source. Name your source as something-doer,
	// instead of do-something.
	Type() string

	// Singleton defines if this type of Source will only be initialized once.
	// For example, if Singleton is true, all Source's in config with the same
	// type will share the same instance (New will be called only once)
	// and Init will be called multiple times for each Source of the same type.
	// Otherwise, each Source will have its own instance, i.e., each New for each Init.
	Singleton() bool
}

// SourceMeta is what users type in their configurations, specifying what source
// they want to use and what properties they provided.
type SourceMeta struct {
	// Type is the name (identifier) of this Source.
	Type string `json:"type"`

	// Properties are user-provided parameters. You should parse it yourself.
	Properties map[string]interface{} `json:"properties"`
}
