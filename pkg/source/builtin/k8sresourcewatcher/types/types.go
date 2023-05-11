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
	"encoding/json"
	"strings"

	"github.com/kubevela/pkg/util/slices"
)

// Config is the config for resource controller
type Config struct {
	APIVersion     string            `json:"apiVersion"`
	Kind           string            `json:"kind"`
	Namespace      string            `json:"namespace,omitempty"`
	Events         []EventType       `json:"events,omitempty"`
	MatchingLabels map[string]string `json:"matchingLabels,omitempty"`
	Clusters       []string          `json:"clusters,omitempty"`
}

// Key returns the identifier of a Config.
func (c *Config) Key() string {
	var labels string
	if len(c.MatchingLabels) > 0 {
		if b, err := json.Marshal(c.MatchingLabels); err == nil {
			labels = string(b)
		}
	}
	return strings.Join([]string{c.APIVersion, c.Kind, c.Namespace, labels}, "-")
}

// Merge merges two Configs.
func (c *Config) Merge(new Config) {
	for _, event := range new.Events {
		if !slices.Contains(c.Events, event) {
			c.Events = append(c.Events, event)
		}
	}
	for k, v := range new.MatchingLabels {
		// no override
		if _, ok := c.MatchingLabels[k]; !ok {
			c.MatchingLabels[k] = v
		}
	}
}

// EventType is the type of the observed event.
type EventType string

// EventTypes
const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
)

// Event represent an event got from k8s api server
type Event struct {
	Type    EventType `json:"type"`
	Cluster string    `json:"cluster"`
}

// InformerEvent indicate the informerEvent
type InformerEvent struct {
	Event
	EventObj interface{}
}
