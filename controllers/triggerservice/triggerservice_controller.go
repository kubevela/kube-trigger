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
	"reflect"

	standardv1alpha1 "github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/controllers/config"
	"github.com/kubevela/kube-trigger/controllers/triggerinstance"
	"github.com/kubevela/kube-trigger/controllers/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles a TriggerService object.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.Config
}

var (
	logger           = logrus.WithField("controller", "kube-trigger-config")
	defaultExtension = ".json"
)

var (
	ErrNoTriggerInstanceSelected = errors.New("no TriggerInstance selected. Check your spec.selector")
)

//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggerconfigs/finalizers,verbs=update

//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers,verbs=get;list
//+kubebuilder:rbac:groups=standard.oam.dev,resources=kubetriggers/status,verbs=get

//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TriggerService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ktc := standardv1alpha1.TriggerService{}
	if err := r.Get(ctx, req.NamespacedName, &ktc); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch TriggerService CRD")
		return ctrl.Result{}, err
	}
	logger.Infof("received reconcile request: %s", req.String())

	kil := standardv1alpha1.TriggerInstanceList{}
	var labelMatcher client.MatchingLabels = ktc.Spec.Selector

	// If the selector provided in the TriggerService is empty
	// and the user want to use the default instance,
	// give the user the default TriggerInstance.
	//
	// Note that if the user have provided selectors, but it does not
	// select anything. It is considered invalid. Don't provide the
	// default TriggerInstance.
	if len(labelMatcher) == 0 && r.Config.ServiceUseDefaultInstance {
		defaultObjKey := types.NamespacedName{
			Namespace: triggerinstance.DefaultInstanceNamespace,
			Name:      triggerinstance.DefaultInstanceName,
		}
		logger.Warnf("no TriggerInstance selected in %s, the default one %s will be used instead",
			req.String(), defaultObjKey.String())
		defaultInstance := standardv1alpha1.TriggerInstance{}
		err := r.Get(ctx, defaultObjKey, &defaultInstance)
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "cannot get the deault instance")
		}
		kil.Items = append(kil.Items, defaultInstance)
	} else {
		var listOptions []client.ListOption
		listOptions = append(listOptions, client.InNamespace(ktc.Namespace), labelMatcher)
		if err := r.List(ctx, &kil, listOptions...); err != nil {
			return ctrl.Result{}, err
		}
	}

	if len(kil.Items) == 0 {
		// TODO(charlie0129): check this error in validation webhook. Notify the user even before TriggerService is applied.
		logger.Error(ErrNoTriggerInstanceSelected)
		return ctrl.Result{}, ErrNoTriggerInstanceSelected
	}

	for _, kt := range kil.Items {
		err := r.addOrDeleteConfigToKubeTrigger(ctx, ktc, kt, req)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) addOrDeleteConfigToKubeTrigger(
	ctx context.Context,
	ks standardv1alpha1.TriggerService,
	ki standardv1alpha1.TriggerInstance,
	req ctrl.Request,
) error {
	// Find TriggerInstance ConfigMap
	cm := v1.ConfigMap{}
	var foundCM bool
	for _, res := range ki.Status.CreatedResources {
		if res.APIVersion != v1.SchemeGroupVersion.String() ||
			res.Kind != reflect.TypeOf(v1.ConfigMap{}).Name() {
			continue
		}
		foundCM = true
		err := r.Get(ctx, types.NamespacedName{
			Namespace: res.Namespace,
			Name:      res.Name,
		}, &cm)
		if err != nil {
			return err
		}
		break
	}
	if !foundCM {
		return fmt.Errorf("no ConfigMap found in TriggerInstance: %s", utils.GetNamespacedName(&ki))
	}

	// Add TriggerService into ConfigMap
	jsonByte, err := json.Marshal(ks.Spec)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal watchers in %s", utils.GetNamespacedName(&ks))
	}

	keyName := req.Name + defaultExtension
	// Indicates if the data in cm is actually changed.
	updated := true

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	if ks.GetUID() == "" {
		logger.Infof("deleted config entry %s from cm %s", keyName, utils.GetNamespacedName(&cm))
		delete(cm.Data, keyName)
	} else {
		if _, ok := cm.Data[keyName]; ok {
			logger.Warnf("key %s already exists in cm %s, will be overwritten", keyName, utils.GetNamespacedName(&cm))
		}
		logger.Infof("added config entry %s to cm %s", keyName, utils.GetNamespacedName(&cm))
		newCfg := string(jsonByte)
		if cm.Data[keyName] == newCfg {
			updated = false
		} else {
			cm.Data[keyName] = newCfg
		}
	}

	// Only if the CM is actually changed, then we need to update cm and restart
	// trigger instance.
	if updated {
		err = r.Update(ctx, &cm)
		if err != nil {
			return err
		}
		return r.restartPod(ctx, ki)
	}

	return nil
}

func (r *Reconciler) restartPod(
	ctx context.Context,
	ki standardv1alpha1.TriggerInstance,
) error {
	var err error

	pods := v1.PodList{}
	err = r.List(ctx, &pods, client.InNamespace(ki.Namespace), client.MatchingLabels{
		triggerinstance.NameLabel: ki.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "cannot list pods")
	}

	for _, pod := range pods.Items {
		err = r.Delete(ctx, pod.DeepCopy())
		logger.Infof("restrting TriggerInstance %s due to config changes", utils.GetNamespacedName(&ki))
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
