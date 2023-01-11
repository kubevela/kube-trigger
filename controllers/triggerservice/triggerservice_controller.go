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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/config"
	"github.com/kubevela/kube-trigger/controllers/triggerinstance"
)

// Reconciler reconciles a TriggerService object.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.Config
}

var (
	logger                  = logrus.WithField("controller", "trigger-service")
	triggerServiceFinalizer = "trigger.oam.dev/trigger-service-finalizer"
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

	ti := &standardv1alpha1.TriggerInstance{}
	if ts.Spec.InstanceRef != "" {
		if err := r.Get(ctx, types.NamespacedName{
			Name:      ts.Spec.InstanceRef,
			Namespace: ts.Namespace,
		}, ti); err != nil {
			logger.Error(err, "failed to get TriggerInstance", "name", ts.Spec.InstanceRef, "namespace", req.Namespace)
			return ctrl.Result{}, err
		}
	} else {
		// use TriggerInstance with the same name as the TriggerService, if not exists, create one
		if err := r.Get(ctx, req.NamespacedName, ti); err != nil {
			if apierrors.IsNotFound(err) {
				ti = &standardv1alpha1.TriggerInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      req.Name,
						Namespace: req.Namespace,
					},
				}
				if err := r.Create(ctx, ti); err != nil {
					logger.Error(err, "failed to create TriggerInstance", "name", ts.Spec.InstanceRef, "namespace", req.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				logger.Error(err, "failed to get TriggerInstance", "name", ts.Spec.InstanceRef, "namespace", req.Namespace)
				return ctrl.Result{}, err
			}
		}
	}

	err := r.handleTriggerConfig(ctx, ts, ti)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) handleTriggerConfig(ctx context.Context, ts *standardv1alpha1.TriggerService, ti *standardv1alpha1.TriggerInstance) error {
	// Add TriggerService into ConfigMap
	jsonByte, err := json.Marshal(ts.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal TriggerService %s: %w", ts.Name, err)
	}
	// Find TriggerInstance ConfigMap
	cm := &v1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ti.Namespace, Name: ti.Name}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			cm = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ti.Name,
					Namespace: ti.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         ti.APIVersion,
							Kind:               ti.Kind,
							Name:               ti.Name,
							UID:                ti.UID,
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					},
				},
				Data: map[string]string{
					ts.Name: string(jsonByte),
				},
			}
			if err := r.Create(ctx, cm); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("failed to get TriggerInstance ConfigMap: %w", err)
		}
	}
	if meta.FinalizerExists(ts, triggerServiceFinalizer) {
		delete(cm.Data, ts.Name)
		if err := r.Update(ctx, cm); err != nil {
			return err
		}
		logger.Infof("delete config entry %s from cm %s", ts.Name, ti.Name)
		meta.RemoveFinalizer(ts, triggerServiceFinalizer)
		return r.Update(ctx, ts)
	}
	if data, ok := cm.Data[ts.Name]; ok {
		if data == string(jsonByte) {
			return nil
		}
		logger.Warnf("key %s already exists in cm %s, will be overwritten", ts.Name, ti.Name)
	}
	cm.Data[ts.Name] = string(jsonByte)
	logger.Infof("add config entry %s from cm %s", ts.Name, ti.Name)
	if err := r.Update(ctx, cm); err != nil {
		return err
	}
	return r.restartPod(ctx, ti)
}

func (r *Reconciler) restartPod(ctx context.Context, ti *standardv1alpha1.TriggerInstance) error {
	var err error

	pods := v1.PodList{}
	err = r.List(ctx, &pods, client.InNamespace(ti.Namespace), client.MatchingLabels{
		triggerinstance.NameLabel: ti.Name,
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
