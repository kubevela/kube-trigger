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

	ki := v1alpha1.TriggerInstance{}
	kiJSON, _ := yaml.YAMLToJSON([]byte(normalTriggerInstance))

	BeforeEach(func() {
		Expect(json.Unmarshal(kiJSON, &ki)).Should(BeNil())
	})

	AfterAll(func() {
		Expect(k8sClient.Delete(ctx, ki.DeepCopy()))
	})

	It("test normal triggerInstance create relevant resource", func() {
		kiKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}
		Expect(k8sClient.Create(ctx, ki.DeepCopy())).Should(BeNil())
		Expect(util.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: kiKey})).Should(BeNil())

		newKi := v1alpha1.TriggerInstance{}
		Expect(k8sClient.Get(ctx, kiKey, &newKi)).Should(BeNil())
		Expect(newKi.Name).Should(Equal("kubetrigger-test"))
		Expect(len(newKi.Status.CreatedResources)).Should(Equal(4))
		for _, createdResource := range newKi.Status.CreatedResources {
			Expect(createdResource.Name).Should(Equal(newKi.Name))
		}

		cm := corev1.ConfigMap{}
		cmKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}

		Expect(k8sClient.Get(ctx, cmKey, &cm)).Should(BeNil())
		Expect(len(cm.OwnerReferences)).Should(Equal(1))
		Expect(cm.OwnerReferences[0].Name).Should(Equal(newKi.Name))
		Expect(cm.OwnerReferences[0].UID).Should(Equal(newKi.UID))

		clusterRoleBing := rbacv1.ClusterRoleBinding{}
		crbKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}
		Expect(k8sClient.Get(ctx, crbKey, &clusterRoleBing)).Should(BeNil())
		Expect(len(clusterRoleBing.Subjects)).Should(Equal(1))
		Expect(clusterRoleBing.Subjects[0].Name).Should(Equal(newKi.Name))
		Expect(clusterRoleBing.Subjects[0].Namespace).Should(Equal(newKi.Namespace))

		sa := corev1.ServiceAccount{}
		saKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}
		Expect(k8sClient.Get(ctx, saKey, &sa)).Should(BeNil())
		Expect(len(sa.OwnerReferences)).Should(Equal(1))
		Expect(sa.OwnerReferences[0].Name).Should(Equal(newKi.Name))
		Expect(sa.OwnerReferences[0].UID).Should(Equal(newKi.UID))

		deploy := appsv1.Deployment{}
		deployKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}
		Expect(k8sClient.Get(ctx, deployKey, &deploy)).Should(BeNil())
		Expect(len(deploy.OwnerReferences)).Should(Equal(1))
		Expect(deploy.OwnerReferences[0].Name).Should(Equal(newKi.Name))
		Expect(deploy.OwnerReferences[0].UID).Should(Equal(newKi.UID))
		Expect(deploy.Spec.Template.Spec.ServiceAccountName).Should(Equal(newKi.Name))
		Expect(deploy.Spec.Template.Spec.Volumes[0].ConfigMap.Name).Should(Equal(newKi.Name))
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
