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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubeTriggerConfigSpec defines the desired state of KubeTriggerConfig
type KubeTriggerConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Selector map[string]string `json:"selector"`
	// Config for kube-trigger
	//+optional
	Watchers []Watcher `json:"watchers"`
}

type Watcher struct {
	Source  Meta   `json:"source"`
	Filters []Meta `json:"filters"`
	Actions []Meta `json:"actions"`
}

type Meta struct {
	Type string `json:"type"`
	//+kubebuilder:pruning:PreserveUnknownFields
	Properties *runtime.RawExtension `json:"properties"`
}

// KubeTriggerConfigStatus defines the observed state of KubeTriggerConfig
type KubeTriggerConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KubeTriggerConfig is the Schema for the kubetriggerconfigs API
type KubeTriggerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeTriggerConfigSpec   `json:"spec,omitempty"`
	Status KubeTriggerConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KubeTriggerConfigList contains a list of KubeTriggerConfig
type KubeTriggerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeTriggerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeTriggerConfig{}, &KubeTriggerConfigList{})
}
