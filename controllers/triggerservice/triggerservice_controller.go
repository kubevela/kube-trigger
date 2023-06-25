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
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cuelang.org/go/cue"

	"github.com/kubevela/pkg/cue/cuex"
	"github.com/kubevela/pkg/util/slices"
	"github.com/kubevela/pkg/util/template/definition"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/utils"
	triggertypes "github.com/kubevela/kube-trigger/pkg/types"
)

// Reconciler reconciles a TriggerService object.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	logger = logrus.WithField("controller", "trigger-service")
)

const (
	triggerNameLabel        string = "trigger.oam.dev/name"
	triggerServiceFinalizer string = "trigger.oam.dev/trigger-service-finalizer"
)

//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs/finalizers,verbs=update

//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers,verbs=get;list
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers/status,verbs=get

//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;update

// Reconcile reconciles a TriggerService object.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ts := &standardv1alpha1.TriggerService{}
	logger = logger.WithField("trigger-service", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, ts); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "failed to get TriggerService")
		return ctrl.Result{}, err
	}
	logger.Infof("Reconciling TriggerService %s", req.Name)
	if ts.ObjectMeta.GetDeletionTimestamp() != nil {
		if !meta.FinalizerExists(ts, triggerServiceFinalizer) {
			meta.AddFinalizer(ts, triggerServiceFinalizer)
			return ctrl.Result{}, r.Update(ctx, ts)
		}
	}

	if err := r.handleTriggerConfig(ctx, ts); err != nil {
		logger.Error(err, "failed to handle trigger config")
		return ctrl.Result{}, err
	}

	if err := r.handleWorker(ctx, ts); err != nil {
		logger.Error(err, "failed to handle trigger worker")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) handleWorker(ctx context.Context, ts *standardv1alpha1.TriggerService) error {

	v, err := r.loadTemplateCueValue(ctx, ts)
	if err != nil {
		return err
	}

	if err := r.createRoles(ctx, ts, v); err != nil {
		return err
	}

	if err := r.createDeployment(ctx, ts, v); err != nil {
		return err
	}

	if err := r.createService(ctx, ts, v); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) loadTemplateCueValue(ctx context.Context, ts *standardv1alpha1.TriggerService) (cue.Value, error) {
	templateName := "default"
	opts := make([]cuex.CompileOption, 0)
	opts = append(opts, cuex.WithExtraData("triggerService", map[string]string{
		"name":      ts.Name,
		"namespace": ts.Namespace,
	}))
	if ts.Spec.Worker != nil {
		if ts.Spec.Worker.Template != "" {
			templateName = ts.Spec.Worker.Template
		}
		opts = append(opts, cuex.WithExtraData("parameter", ts.Spec.Worker.Properties))
	}
	template, err := definition.NewTemplateLoader(ctx, r.Client).LoadTemplate(ctx, templateName, definition.WithType(triggertypes.DefinitionTypeTriggerWorker))
	if err != nil {
		return cue.Value{}, err
	}

	v, err := cuex.CompileStringWithOptions(ctx, template.Compile(), opts...)
	if err != nil {
		return cue.Value{}, err
	}

	return v, nil
}

func (r *Reconciler) createDeployment(ctx context.Context, ts *standardv1alpha1.TriggerService, v cue.Value) error {
	data, err := v.LookupPath(cue.ParsePath("deployment")).MarshalJSON()
	if err != nil {
		return err
	}

	expectedDeployment := new(appsv1.Deployment)
	if err := json.Unmarshal(data, expectedDeployment); err != nil {
		return err
	}

	existDeployment := new(appsv1.Deployment)
	if err := r.Get(ctx, types.NamespacedName{Namespace: ts.Namespace, Name: ts.Name}, existDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			utils.SetOwnerReference(expectedDeployment, ts)
			return r.Create(ctx, expectedDeployment)
		}
		return err
	}

	return r.Patch(ctx, expectedDeployment, client.Merge)
}

func (r *Reconciler) createRoles(ctx context.Context, ts *standardv1alpha1.TriggerService, v cue.Value) error {
	saName, err := v.LookupPath(cue.ParsePath("parameter.serviceAccount")).String()
	if err != nil {
		return err
	}
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ts.Namespace, Name: saName}, sa); err != nil {
		if apierrors.IsNotFound(err) {
			sa = &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      saName,
					Namespace: ts.Namespace,
				},
			}
			utils.SetOwnerReference(sa, ts)
			if err := r.Create(ctx, sa); err != nil {
				return err
			}
		}
	} else {
		return err
	}
	role := &rbacv1.ClusterRoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: "kube-trigger"}, role); err != nil {
		return err
	}
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      saName,
		Namespace: ts.Namespace,
	}
	if !slices.Contains(role.Subjects, subject) {
		role.Subjects = append(role.Subjects, subject)
		return r.Update(ctx, role)
	}
	return nil
}

func (r *Reconciler) createService(ctx context.Context, ts *standardv1alpha1.TriggerService, v cue.Value) error {
	needCreateService, err := v.LookupPath(cue.ParsePath("parameter.createService")).Bool()
	if err != nil {
		return err
	}

	if !needCreateService {
		existSvc := new(corev1.Service)
		if err := r.Get(ctx, types.NamespacedName{Namespace: ts.Namespace, Name: ts.Name}, existSvc); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		return r.Delete(ctx, existSvc)
	}

	data, err := v.LookupPath(cue.ParsePath("service")).MarshalJSON()
	if err != nil {
		return err
	}

	expectSvc := new(corev1.Service)
	if err := json.Unmarshal(data, expectSvc); err != nil {
		return err
	}

	existSvc := new(corev1.Service)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ts.Namespace, Name: ts.Name}, existSvc); err != nil {
		if apierrors.IsNotFound(err) {
			utils.SetOwnerReference(expectSvc, ts)
			return r.Client.Create(ctx, expectSvc)
		}
		return err
	}
	return r.Client.Patch(ctx, expectSvc, client.Merge)
}

func (r *Reconciler) handleTriggerConfig(ctx context.Context, ts *standardv1alpha1.TriggerService) error {
	// Add TriggerService into ConfigMap
	jsonByte, err := json.Marshal(ts.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal TriggerService %s: %w", ts.Name, err)
	}
	key := fmt.Sprintf("%s.json", ts.Name)
	// Find TriggerInstance ConfigMap
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ts.Namespace, Name: ts.Name}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ts.Name,
					Namespace: ts.Namespace,
				},
				Data: map[string]string{
					key: string(jsonByte),
				},
			}
			utils.SetOwnerReference(cm, ts)
			return r.Create(ctx, cm)
		}
		return fmt.Errorf("failed to get TriggerService ConfigMap: %w", err)
	}
	cm.Data[key] = string(jsonByte)
	logger.Infof("add config entry %s from cm %s", ts.Name, ts.Name)
	if err := r.Update(ctx, cm); err != nil {
		return err
	}
	return r.restartPod(ctx, ts)
}

func (r *Reconciler) restartPod(ctx context.Context, ts *standardv1alpha1.TriggerService) error {
	var err error

	pods := corev1.PodList{}
	err = r.List(ctx, &pods, client.InNamespace(ts.Namespace), client.MatchingLabels{
		triggerNameLabel: ts.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pods")
	}

	for _, pod := range pods.Items {
		err = r.Delete(ctx, pod.DeepCopy())
		logger.Infof("restarting TriggerInstance due to config changes")
		if err != nil {
			return errors.Wrapf(err, "cannot delete pod: %s/%s", pod.Namespace, pod.Name)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&standardv1alpha1.TriggerService{}).
		Complete(r)
}
