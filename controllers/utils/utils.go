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

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	pointer "k8s.io/utils/ptr"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
)

// SetOwnerReference set owner reference for trigger service.
func SetOwnerReference(obj metav1.Object, ts *standardv1alpha1.TriggerService) {
	obj.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion:         standardv1alpha1.GroupVersion.String(),
		Kind:               standardv1alpha1.TriggerServiceKind,
		Name:               ts.Name,
		UID:                ts.GetUID(),
		BlockOwnerDeletion: pointer.To(true),
	}})
}

// GetNamespacedName get namespaced name from resource.
func GetNamespacedName(obj metav1.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}
