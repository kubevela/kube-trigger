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
	"time"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/config"
	"github.com/kubevela/kube-trigger/pkg/util"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles a TriggerInstance object.
type Reconciler struct {
	client.Client
	StatusWriter client.StatusWriter
	Scheme       *runtime.Scheme
	Config       config.Config
}

var (
	logger = logrus.WithField("controller", "kube-trigger")
	// defaultInstanceLastCreatedTime is used to make sure the existence of default instance is not constantly checked.
	defaultInstanceLastCreatedTime = time.Time{}
)

const (
	DefaultInstanceName                       = "default"
	DefaultInstanceNamespace                  = "kube-trigger-system"
	InstanceNameLabel                         = "instance"
	MinIntervalBetweenCheckingDefaultInstance = 5
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

	// Create relevant resource of ki.
	err := r.createRelevantResources(ctx, &ki, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	if r.Config.CreateDefaultInstance {
		r.createDefaultInstance(ctx)
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) createRelevantResources(
	ctx context.Context,
	ki *standardv1alpha1.TriggerInstance,
	req ctrl.Request,
) error {
	if err := r.ReconcileClusterRoleBinding(ctx, ki, req); err != nil {
		logger.Errorf("reconcile ClusterRoleBinding failed: %s", err)
		return err
	}

	if err := r.ReconcileServiceAccount(ctx, ki, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return err
	}

	if err := r.ReconcileConfigMap(ctx, ki, req); err != nil {
		logger.Errorf("reconcile ServiceAccount failed: %s", err)
		return err
	}

	if err := r.ReconcileDeployment(ctx, ki, req); err != nil {
		logger.Errorf("reconcile Deployment failed: %s", err)
		return err
	}

	return nil
}

func (r *Reconciler) createDefaultInstance(ctx context.Context) {
	// Only proceed if it is at least 5s after the last run to prevent constant re-runs
	// and the default instance is unlikely to fail.
	if !time.Now().Add(time.Second * time.Duration(MinIntervalBetweenCheckingDefaultInstance)).
		After(defaultInstanceLastCreatedTime) {
		return
	}
	// Record the time this run whether the rest logic is successful or not.
	defaultInstanceLastCreatedTime = time.Now()

	defaultInstance := standardv1alpha1.TriggerInstance{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: DefaultInstanceNamespace,
		Name:      DefaultInstanceName,
	}, &defaultInstance)
	if err == nil {
		return
	}
	if util.IgnoreNotFound(err) != nil {
		logger.Errorf("getting default instance failed, the default TriggerInstance may not be created: %s", err)

		// Default instance not found, create a default one.
		// TODO(charlie0129): also check its status and spec, if not right, fix it.
		defaultInstance.Namespace = DefaultInstanceNamespace
		defaultInstance.Name = DefaultInstanceName
		defaultInstance.Labels = map[string]string{
			InstanceNameLabel: DefaultInstanceName,
		}
		err = r.Create(ctx, &defaultInstance)
		if err != nil {
			logger.Errorf("creating default isntance failed when it not found: %s", err)
			return
		}

		logger.Infof("default instance %s/%s with labels %v created",
			defaultInstance.Namespace,
			defaultInstance.Name,
			defaultInstance.Labels)
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO(charlie0129): also listen to other resource events
	return ctrl.NewControllerManagedBy(mgr).
		For(&standardv1alpha1.TriggerInstance{}).
		Complete(r)
}
