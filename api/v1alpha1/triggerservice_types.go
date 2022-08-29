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
	"github.com/kubevela/kube-trigger/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TriggerServiceSpec defines the desired state of TriggerService.
type TriggerServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Selector map[string]string `json:"selector"`
	// Config for kube-trigger
	//+optional
	//+kubebuilder:object:generate=true
	Triggers []config.TriggerMetaWrapper `json:"triggers"`
}

// TriggerServiceStatus defines the observed state of TriggerService.
type TriggerServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// TriggerService is the Schema for the kubetriggerconfigs API.
type TriggerService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerServiceSpec   `json:"spec,omitempty"`
	Status TriggerServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TriggerServiceList contains a list of TriggerService.
type TriggerServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TriggerService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TriggerService{}, &TriggerServiceList{})
}
