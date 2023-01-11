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

package triggerinstance

import (
	"context"
	"encoding/json"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var _ = Describe("TriggerinstanceController", Ordered, func() {
	ctx := context.TODO()

	ti := v1alpha1.TriggerInstance{}
	kiJSON, _ := yaml.YAMLToJSON([]byte(normalTriggerInstance))

	BeforeEach(func() {
		Expect(json.Unmarshal(kiJSON, &ti)).Should(BeNil())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ti.DeepCopy()))
	})

	It("test normal triggerInstance create relevant resource", func() {
		tiKey := client.ObjectKey{
			Namespace: ti.Namespace,
			Name:      ti.Name,
		}
		Expect(k8sClient.Create(ctx, ti.DeepCopy())).Should(BeNil())
		Expect(util.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: tiKey})).Should(BeNil())

		tiObj := &v1alpha1.TriggerInstance{}
		Expect(k8sClient.Get(ctx, tiKey, tiObj)).Should(BeNil())

		clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		Expect(k8sClient.Get(ctx, tiKey, clusterRoleBinding)).Should(BeNil())
		Expect(len(clusterRoleBinding.Subjects)).Should(Equal(1))
		Expect(clusterRoleBinding.Subjects[0].Name).Should(Equal(tiObj.Name))
		Expect(clusterRoleBinding.Subjects[0].Namespace).Should(Equal(tiObj.Namespace))
		Expect(len(clusterRoleBinding.OwnerReferences)).Should(Equal(1))
		Expect(clusterRoleBinding.OwnerReferences[0].Name).Should(Equal(tiObj.Name))
		Expect(clusterRoleBinding.OwnerReferences[0].UID).Should(Equal(tiObj.UID))

		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(ctx, tiKey, sa)).Should(BeNil())
		Expect(len(sa.OwnerReferences)).Should(Equal(1))
		Expect(sa.OwnerReferences[0].Name).Should(Equal(tiObj.Name))
		Expect(sa.OwnerReferences[0].UID).Should(Equal(tiObj.UID))

		deploy := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, tiKey, deploy)).Should(BeNil())
		Expect(len(deploy.OwnerReferences)).Should(Equal(1))
		Expect(deploy.OwnerReferences[0].Name).Should(Equal(tiObj.Name))
		Expect(deploy.OwnerReferences[0].UID).Should(Equal(tiObj.UID))
		Expect(deploy.Spec.Template.Spec.ServiceAccountName).Should(Equal(tiObj.Name))
		Expect(deploy.Spec.Template.Spec.Volumes[0].ConfigMap.Name).Should(Equal(tiObj.Name))
	})
})

const (
	normalTriggerInstance = `
apiVersion: standard.oam.dev/v1alpha1
kind: TriggerInstance
metadata:
  name: kubetrigger-test
  namespace: default
  labels:
    instance: kubetrigger-sample
spec:
`
)
