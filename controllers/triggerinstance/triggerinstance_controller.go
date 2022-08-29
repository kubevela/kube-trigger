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

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles a TriggerInstance object.
type Reconciler struct {
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
// We need to create ClusterRoleBinding, so * is used.
// TODO: use stricter psermissions
//+kubebuilder:rbac:groups=*,resources=*,verbs=*
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;create;update;delete

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logrus.SetLevel(logrus.DebugLevel)

	ki := standardv1alpha1.TriggerInstance{}
	if err := r.Get(ctx, req.NamespacedName, &ki); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch TriggerInstance CRD")
		return ctrl.Result{}, err
	}
	logger.Infof("received reconcile request: %s", req.String())
	logger.Debugf("obj: %#v", ki)

	defer func() {
		if ki.GetUID() == "" {
			return
		}
		err := r.StatusWriter.Update(ctx, &ki)
		logger.Debugf("updated status: %v", ki.Status)
		if err != nil {
			logger.Errorf("cannot update TriggerInstance: %s", err)
		}
	}()

	if err := r.ReconcileClusterRoleBinding(ctx, &ki, req); err != nil {
		logger.Errorf("reconcile ClusterRoleBinding failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileServiceAccount(ctx, &ki, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileConfigMap(ctx, &ki, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return ctrl.Result{}, err
	}

	if err := r.ReconcileDeployment(ctx, &ki, req); err != nil {
		logger.Errorf("reconcile Deployment failed: %s", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO(charlie0129): also listen to other resource events
	return ctrl.NewControllerManagedBy(mgr).
		For(&standardv1alpha1.TriggerInstance{}).
		Complete(r)
}
