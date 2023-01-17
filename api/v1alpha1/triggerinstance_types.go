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
)

// TriggerInstanceSpec defines the desired state of TriggerInstance.
type TriggerInstanceSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Cache size for filters and actions.
	//+optional
	RegistrySize *int `json:"registrySize,omitempty"`

	//+optional
	LogLevel *string `json:"logLevel,omitempty"`

	//+optional
	WorkerConfig *WorkerConfig `json:"workerConfig,omitempty"`

	// TODO(charlie0129): add RBAC config, container image
}

// WorkerConfig defines the worker config
type WorkerConfig struct {
	//+optional
	ActionRetry *bool `json:"actionRetry"`

	// Max retry count after action failed to run.
	//+optional
	//+kubebuilder:validation:Minimum=0
	MaxRetry *int `json:"maxRetry,omitempty"`

	// First delay to retry actions in seconds, subsequent delays will grow exponentially.
	//+optional
	//+kubebuilder:validation:Minimum=0
	RetryDelay *int `json:"retryDelay,omitempty"`

	// Long-term QPS limiting per worker, this is shared between all watchers.
	//+optional
	//+kubebuilder:validation:Minimum=1
	PerWorkerQPS *int `json:"perWorkerQPS,omitempty"`

	// Queue size for running actions, this is shared between all watchers.
	//+optional
	//+kubebuilder:validation:Minimum=0
	QueueSize *int `json:"queueSize,omitempty"`

	// Timeout for each job in seconds.
	//+optional
	//+kubebuilder:validation:Minimum=1
	Timeout *int `json:"timeout,omitempty"`

	// Number of workers for running actions, this is shared between all watchers.
	//+optional
	//+kubebuilder:validation:Minimum=1
	WorkerCount *int `json:"workerCount,omitempty"`
}

// TriggerInstanceStatus defines the observed state of TriggerInstance.
type TriggerInstanceStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	// TODO(charlie0129): add status fields
	// - If a kube-trigger instance is working fine
	// - statistics
	//   - running jobs count
	//   - success jobs count
	//   - failed jobs count
	// TODO: make it useful
	Healthy bool `json:"healthy"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TriggerInstance is the Schema for the kubetriggers API.
type TriggerInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerInstanceSpec   `json:"spec,omitempty"`
	Status TriggerInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TriggerInstanceList contains a list of TriggerInstance.
type TriggerInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TriggerInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TriggerInstance{}, &TriggerInstanceList{})
}
