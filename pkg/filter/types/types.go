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

const (
	// TypeFieldName is type field name in string representation. Will be used
	// when parsing configurations.
	TypeFieldName = "type"

	// PropertiesFieldName is properties field name in string representation.
	// Will be used when parsing configurations.
	PropertiesFieldName = "properties"
)

// Filter is an interface for filters. Anything that implements this interface
// can be registered as filters, and will be called automatically.
type Filter interface {
	// New returns a new uninitialized instance.
	New() Filter

	// Validate validates properties.
	Validate(properties map[string]interface{}) error

	// Init initializes this instance using user-provided properties.
	// Typically, this will only be called once and the initialized instance
	// will be cached.
	Init(properties map[string]interface{}) error

	// ApplyToObject applies this Filter to the given object that came from
	// sources. Returning false will filter this object out. Since this method
	// will be called multiple times, you should make sure it is idempotent.
	//
	// The string in return values is used to pass messages. For example,
	// your filter gets a pod failure event, and you want to say "hey this pod failed".
	// You can put that in the string, and it can be passed to Actions. Actions
	// can, for example, send this message to webhooks.
	ApplyToObject(event interface{}, data interface{}) (bool, string, error)

	// Type returns the type of this Filter. Name your filter as something-doer,
	// instead of do-something.
	Type() string
}

// FilterMeta is what users type in their configurations, specifying what filter
// they want to use and what properties they provided.
type FilterMeta struct {
	// Type is the name (identifier) of this filter.
	Type string `json:"type"`

	// Properties are user-provided parameters. You should parse it yourself.
	Properties map[string]interface{} `json:"properties"`

	// Raw is the raw string representation of this filter. Typically, you will
	// not use it. This is for identifying filter instances.
	Raw string `json:"raw,omitempty"`
}
