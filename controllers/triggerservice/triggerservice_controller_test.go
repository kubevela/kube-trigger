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

package triggerservice

import (
	"context"
	"encoding/json"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/pkg/util/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/utils"
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

	ts := v1alpha1.TriggerService{}
	tsJSON, _ := yaml.YAMLToJSON([]byte(normalTriggerInstance))

	tsNoService := v1alpha1.TriggerService{}
	tsNoServiceJSON, _ := yaml.YAMLToJSON([]byte(normalTriggerServiceWithOutService))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vela-system",
		},
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-trigger",
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	BeforeEach(func() {
		Expect(k8sClient.Create(ctx, ns.DeepCopy())).Should(SatisfyAny(Succeed(), &utils.AlreadyExistMatcher{}))
		Expect(k8sClient.Create(ctx, clusterRoleBinding.DeepCopy())).Should(SatisfyAny(Succeed(), &utils.AlreadyExistMatcher{}))
		for _, file := range []string{"bump-application-revision", "create-event-listener", "default", "patch-resource", "record-event"} {
			Expect(utils.InstallDefinition(ctx, k8sClient, filepath.Join("../../config/definition", file+".yaml"))).
				Should(SatisfyAny(Succeed(), &utils.AlreadyExistMatcher{}))
		}
		Expect(json.Unmarshal(tsJSON, &ts)).Should(BeNil())
		Expect(json.Unmarshal(tsNoServiceJSON, &tsNoService)).Should(BeNil())
	})

	It("test normal triggerInstance create relevant resource", func() {
		tsKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}
		Expect(k8sClient.Create(ctx, ts.DeepCopy())).Should(BeNil())
		Expect(utils.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: tsKey})).Should(BeNil())

		newTs := v1alpha1.TriggerService{}
		Expect(k8sClient.Get(ctx, tsKey, &newTs)).Should(BeNil())
		Expect(newTs.Name).Should(Equal("kubetrigger-sample-config-service"))

		cm := corev1.ConfigMap{}
		cmKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}

		Expect(k8sClient.Get(ctx, cmKey, &cm)).Should(BeNil())
		Expect(len(cm.OwnerReferences)).Should(Equal(1))
		Expect(cm.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(cm.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		clusterRoleBing := rbacv1.ClusterRoleBinding{}
		crbKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      "kube-trigger",
		}
		Expect(k8sClient.Get(ctx, crbKey, &clusterRoleBing)).Should(BeNil())
		subject := rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "kube-trigger",
			Namespace: ts.Namespace,
		}
		Expect(slices.Contains(clusterRoleBing.Subjects, subject)).Should(BeTrue())

		sa := corev1.ServiceAccount{}
		saKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      "kube-trigger",
		}
		Expect(k8sClient.Get(ctx, saKey, &sa)).Should(BeNil())
		Expect(len(sa.OwnerReferences)).Should(Equal(1))
		Expect(sa.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(sa.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		deploy := appsv1.Deployment{}
		deployKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}
		Expect(k8sClient.Get(ctx, deployKey, &deploy)).Should(BeNil())
		Expect(len(deploy.OwnerReferences)).Should(Equal(1))
		Expect(deploy.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(deploy.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		svc := corev1.Service{}
		svcKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}
		Expect(k8sClient.Get(ctx, svcKey, &svc)).Should(BeNil())
		Expect(len(svc.OwnerReferences)).Should(Equal(1))
		Expect(svc.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(svc.OwnerReferences[0].UID).Should(Equal(newTs.UID))
	})

	It("test normal triggerInstance create relevant resource, update config and restart pod", func() {
		tsKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}

		newTs := v1alpha1.TriggerService{}
		Expect(k8sClient.Get(ctx, tsKey, &newTs)).Should(BeNil())
		Expect(newTs.Name).Should(Equal("kubetrigger-sample-config-service"))

		triggers := newTs.Spec.Triggers
		for i := range triggers {
			bytes, err := triggers[i].Source.Properties.MarshalJSON()
			if err != nil {
				return
			}
			conf := new(types.Config)
			err = json.Unmarshal(bytes, conf)

			conf.Events = append(conf.Events, types.EventTypeCreate)
			marshal, err := json.Marshal(conf)
			extension := &runtime.RawExtension{Raw: marshal}
			triggers[i].Source.Properties = extension
		}

		Expect(k8sClient.Update(ctx, &newTs)).Should(BeNil())
		Expect(utils.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: tsKey})).Should(BeNil())

		cm := corev1.ConfigMap{}
		cmKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}

		Expect(k8sClient.Get(ctx, cmKey, &cm)).Should(BeNil())
		Expect(len(cm.OwnerReferences)).Should(Equal(1))
		Expect(cm.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(cm.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		clusterRoleBing := rbacv1.ClusterRoleBinding{}
		crbKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      "kube-trigger",
		}
		Expect(k8sClient.Get(ctx, crbKey, &clusterRoleBing)).Should(BeNil())
		subject := rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "kube-trigger",
			Namespace: ts.Namespace,
		}
		Expect(slices.Contains(clusterRoleBing.Subjects, subject)).Should(BeTrue())

		sa := corev1.ServiceAccount{}
		saKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      "kube-trigger",
		}
		Expect(k8sClient.Get(ctx, saKey, &sa)).Should(BeNil())
		Expect(len(sa.OwnerReferences)).Should(Equal(1))
		Expect(sa.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(sa.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		deploy := appsv1.Deployment{}
		deployKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}
		Expect(k8sClient.Get(ctx, deployKey, &deploy)).Should(BeNil())
		Expect(len(deploy.OwnerReferences)).Should(Equal(1))
		Expect(deploy.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(deploy.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		svc := corev1.Service{}
		svcKey := client.ObjectKey{
			Namespace: ts.Namespace,
			Name:      ts.Name,
		}
		Expect(k8sClient.Get(ctx, svcKey, &svc)).Should(BeNil())
		Expect(len(svc.OwnerReferences)).Should(Equal(1))
		Expect(svc.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(svc.OwnerReferences[0].UID).Should(Equal(newTs.UID))
	})

	It("test normal triggerInstance create relevant resource without service", func() {
		tsNoServiceKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      tsNoService.Name,
		}
		Expect(k8sClient.Create(ctx, tsNoService.DeepCopy())).Should(BeNil())
		Expect(utils.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: tsNoServiceKey})).Should(BeNil())

		newTs := v1alpha1.TriggerService{}
		Expect(k8sClient.Get(ctx, tsNoServiceKey, &newTs)).Should(BeNil())
		Expect(newTs.Name).Should(Equal("kubetrigger-sample-config-no-service"))

		cm := corev1.ConfigMap{}
		cmKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      tsNoService.Name,
		}

		Expect(k8sClient.Get(ctx, cmKey, &cm)).Should(BeNil())
		Expect(len(cm.OwnerReferences)).Should(Equal(1))
		Expect(cm.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(cm.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		clusterRoleBing := rbacv1.ClusterRoleBinding{}
		crbKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      "kube-trigger",
		}
		Expect(k8sClient.Get(ctx, crbKey, &clusterRoleBing)).Should(BeNil())
		subject := rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "kube-trigger",
			Namespace: ts.Namespace,
		}
		Expect(slices.Contains(clusterRoleBing.Subjects, subject)).Should(BeTrue())

		sa := corev1.ServiceAccount{}
		saKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      "no-service",
		}
		Expect(k8sClient.Get(ctx, saKey, &sa)).Should(BeNil())
		Expect(len(sa.OwnerReferences)).Should(Equal(1))
		Expect(sa.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(sa.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		deploy := appsv1.Deployment{}
		deployKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      tsNoService.Name,
		}
		Expect(k8sClient.Get(ctx, deployKey, &deploy)).Should(BeNil())
		Expect(len(deploy.OwnerReferences)).Should(Equal(1))
		Expect(deploy.OwnerReferences[0].Name).Should(Equal(newTs.Name))
		Expect(deploy.OwnerReferences[0].UID).Should(Equal(newTs.UID))

		svc := corev1.Service{}
		svcKey := client.ObjectKey{
			Namespace: tsNoService.Namespace,
			Name:      tsNoService.Name,
		}
		Expect(apierrors.IsNotFound(k8sClient.Get(ctx, svcKey, &svc))).Should(BeTrue())

	})

})

const (
	normalTriggerInstance = `
apiVersion: standard.oam.dev/v1alpha1
kind: TriggerService
metadata:
  name: kubetrigger-sample-config-service
  namespace: default
spec:
  # A trigger is a group of Source, Filters, and Actions.
  # You can add multiple triggers.
  worker:
    properties:
      createService: true
  triggers:
    - source:
        type: resource-watcher
        properties:
          # We are interested in ConfigMap events.
          apiVersion: "v1"
          kind: ConfigMap
          namespace: default
          # Only watch update event.
          events:
            - update
      filter: |
        context: data: metadata: name: =~"this-will-trigger-update-.*"
      action:
        # Bump Application Revision to update Application.
        type: bump-application-revision
        properties:
          namespace: default
          # Select Applications to bump using labels.
          nameSelector:
            fromLabel: "watch-this"
`
	normalTriggerServiceWithOutService = `
apiVersion: standard.oam.dev/v1alpha1
kind: TriggerService
metadata:
  name: kubetrigger-sample-config-no-service
  namespace: default
spec:
  # A trigger is a group of Source, Filters, and Actions.
  # You can add multiple triggers.
  worker:
    properties:
      createService: false
      serviceAccount: no-service
  triggers:
    - source:
        type: resource-watcher
        properties:
          # We are interested in ConfigMap events.
          apiVersion: "v1"
          kind: ConfigMap
          namespace: default
          # Only watch update event.
          events:
            - update
      filter: |
        context: data: metadata: name: =~"this-will-trigger-update-.*"
      action:
        # Bump Application Revision to update Application.
        type: bump-application-revision
        properties:
          namespace: default
          # Select Applications to bump using labels.
          namSelector:
            fromLabel: "watch-this"
`
)
