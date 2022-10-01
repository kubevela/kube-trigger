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

package event

// Event represent an event got from k8s api server
type Event struct {
	EventType  string
	Namespace  string
	Kind       string
	ApiVersion string
	Name       string
	Info       string
}

// Message returns event message in standard format.
func (e *Event) Message() (msg string) {
	return "apiVersion: " + e.ApiVersion + ", kind: " + e.Kind + ", namespace: " + e.Namespace + ", name: " + e.Name + ", info: " + e.Info
}

// InformerEvent indicate the informerEvent
type InformerEvent struct {
	Key          string
	EventType    string
	Namespace    string
	ResourceType string
	EventObj     interface{}
}
