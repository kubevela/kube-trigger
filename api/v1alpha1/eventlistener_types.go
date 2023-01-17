/*
Copyright 2023 The KubeVela Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true

// EventListener is the schema for the event listener.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EventListener struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +nullable
	Events []Event `json:"events,omitempty"`
}

// +kubebuilder:object:root=true

// EventListenerList contains a list of EventListener.
type EventListenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventListener `json:"items"`
}

// Event is the schema for the event.
type Event struct {
	// Resource is the resource that triggers the event.
	Resource EventResource `json:"resource"`
	// Timestamp is the time when the event is triggered.
	Timestamp metav1.Time `json:"timestamp"`
	// Type is the type of the event.
	Type string `json:"type,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// Data is the data of the event that carries.
	Data *runtime.RawExtension `json:"data,omitempty"`
}

// EventResource is the resource that triggers the event.
type EventResource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

func init() {
	SchemeBuilder.Register(&EventListener{}, &EventListenerList{})
}
