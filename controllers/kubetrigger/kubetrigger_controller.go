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

package kubetrigger

import (
	"context"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeTriggerReconciler reconciles a KubeTrigger object.
type KubeTriggerReconciler struct {
	client.Client
	StatusWriter client.StatusWriter
	Scheme       *runtime.Scheme
}

var (
	logger = logrus.WithField("controller", "kube-trigger")
)

//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups=,resources=serviceaccounts,verbs=get;list;create;update;delete

func (r *KubeTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logrus.SetLevel(logrus.DebugLevel)

	kt := standardv1alpha1.KubeTrigger{}
	if err := r.Get(ctx, req.NamespacedName, &kt); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch KubeTrigger CRD")
		return ctrl.Result{}, err
	}
	logger.Infof("received reconcile request: %s", req.String())
	logger.Debugf("obj: %#v", kt)

	defer func() {
		if kt.GetUID() == "" {
			return
		}
		err := r.StatusWriter.Update(ctx, &kt)
		logger.Debugf("updated status: %v", kt.Status)
		if err != nil {
			logger.Errorf("cannot update KubeTrigger: %s", err)
		}
	}()

	if err := r.ReconcileClusterRoleBinding(ctx, &kt, req); err != nil {
		logger.Errorf("reconcile ClusterRoleBinding failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileServiceAccount(ctx, &kt, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileConfigMap(ctx, &kt, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileDeployment(ctx, &kt, req); err != nil {
		logger.Errorf("reconcile Deployment failed: %s", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO(charlie0129): also listen to other resource events
	return ctrl.NewControllerManagedBy(mgr).
		For(&standardv1alpha1.KubeTrigger{}).
		Complete(r)
}
