package triggerinstance

import (
	"context"
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("TriggerinstanceController", Ordered, func() {
	ctx := context.TODO()

	ki := v1alpha1.TriggerInstance{}
	kiJson, _ := yaml.YAMLToJSON([]byte(normalTriggerInstance))

	BeforeEach(func() {
		Expect(json.Unmarshal(kiJson, &ki)).Should(BeNil())
	})

	It("test normal triggerInstance create relevant resource", func() {
		kiKey := client.ObjectKey{
			Namespace: ki.Namespace,
			Name:      ki.Name,
		}
		Expect(k8sClient.Create(ctx, ki.DeepCopy())).Should(BeNil())
		util.ReconcileOnce(reconciler, reconcile.Request{NamespacedName: kiKey})

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
