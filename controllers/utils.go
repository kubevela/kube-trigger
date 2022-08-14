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

package controllers

import (
	"time"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func setOwnerReference(obj metav1.Object, kt standardv1alpha1.KubeTrigger) {
	t := true
	ownerReference := metav1.OwnerReference{
		APIVersion: standardv1alpha1.GroupVersion.String(),
		Kind:       standardv1alpha1.KubeTriggerKind,
		Name:       kt.Name,
		UID:        kt.GetUID(),
		Controller: &t,
	}

	obj.SetOwnerReferences([]metav1.OwnerReference{ownerReference})
}

func getNamespacedName(kt standardv1alpha1.KubeTrigger) types.NamespacedName {
	return types.NamespacedName{
		Namespace: kt.Namespace,
		Name:      kt.Name,
	}
}

func updateResource(kt *standardv1alpha1.KubeTrigger, res standardv1alpha1.Resource) {
	if kt == nil {
		return
	}

	res.LastUpdateTime = metav1.NewTime(time.Now())

	var newCRs []standardv1alpha1.Resource
	var haveCR bool

	for _, cr := range kt.Status.CreatedResources {
		if cr.Kind == res.Kind &&
			cr.APIVersion == res.APIVersion &&
			cr.Namespace == res.Namespace &&
			cr.Name == res.Name {
			newCRs = append(newCRs, res)
			haveCR = true
			continue
		}
		newCRs = append(newCRs, cr)
	}

	if !haveCR {
		newCRs = append(newCRs, res)
	}

	kt.Status.CreatedResources = newCRs
}
