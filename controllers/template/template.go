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

package template

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:generate ../../hack/generate-go-const-from-file.sh yaml/clusterrolebinding.yaml clusterrolebindingTemplate clusterrolebinding
//go:generate ../../hack/generate-go-const-from-file.sh yaml/cm.yaml cmTemplate cm
//go:generate ../../hack/generate-go-const-from-file.sh yaml/deployment.yaml deploymentTemplate deployment
//go:generate ../../hack/generate-go-const-from-file.sh yaml/sa.yaml saTemplate sa

var clusterRoleBinding rbacv1.ClusterRoleBinding
var configMap corev1.ConfigMap
var deployment appsv1.Deployment
var serviceAccount corev1.ServiceAccount

//nolint:gochecknoinits
func init() {
	utilruntime.Must(yaml.Unmarshal([]byte(clusterrolebindingTemplate), &clusterRoleBinding))
	if len(clusterRoleBinding.Subjects) != 1 {
		panic("ClusterRoleBinding must have one subject")
	}

	utilruntime.Must(yaml.Unmarshal([]byte(cmTemplate), &configMap))

	utilruntime.Must(yaml.Unmarshal([]byte(deploymentTemplate), &deployment))
	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		panic("Deployment must have at least one container")
	}
	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		panic("Deployment must have one volume")
	}

	utilruntime.Must(yaml.Unmarshal([]byte(saTemplate), &serviceAccount))
}

func GetClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return clusterRoleBinding.DeepCopy()
}

func GetConfigMap() *corev1.ConfigMap {
	return configMap.DeepCopy()
}

func GetDeployment() *appsv1.Deployment {
	return deployment.DeepCopy()
}

func GetServiceAccount() *corev1.ServiceAccount {
	return serviceAccount.DeepCopy()
}
