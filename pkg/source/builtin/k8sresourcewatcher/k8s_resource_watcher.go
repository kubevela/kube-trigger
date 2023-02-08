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

package k8sresourcewatcher

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/controller"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
)

const (
	clusterLabel    string = "cluster.core.oam.dev/cluster-credential-type"
	defaultCluster  string = "local"
	clusterCertKey  string = "tls.key"
	clusterCertData string = "tls.crt"
	clusterCAData   string = "ca.crt"
	clusterEndpoint string = "endpoint"
)

type K8sResourceWatcher struct {
	configs       map[string]*types.Config
	eventHandlers map[string][]eventhandler.EventHandler
	logger        *logrus.Entry
}

var _ sourcetypes.Source = &K8sResourceWatcher{}

func (w *K8sResourceWatcher) New() sourcetypes.Source {
	return &K8sResourceWatcher{
		configs:       make(map[string]*types.Config),
		eventHandlers: make(map[string][]eventhandler.EventHandler),
	}
}

func (w *K8sResourceWatcher) Parse(properties *runtime.RawExtension) (*types.Config, error) {
	props, err := properties.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ctrlConf := &types.Config{}
	err = json.Unmarshal(props, ctrlConf)
	if err != nil {
		return nil, errors.Wrapf(err, "error when parsing properties for %s", w.Type())
	}
	return ctrlConf, nil
}

func (w *K8sResourceWatcher) Init(properties *runtime.RawExtension, eh eventhandler.EventHandler) error {
	var err error

	props, err := properties.MarshalJSON()
	if err != nil {
		return err
	}
	ctrlConf := &types.Config{}
	err = json.Unmarshal(props, ctrlConf)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties for %s", w.Type())
	}
	if orig, ok := w.configs[ctrlConf.Key()]; ok {
		orig.Merge(*ctrlConf)
		w.configs[ctrlConf.Key()] = orig
	} else {
		w.configs[ctrlConf.Key()] = ctrlConf
	}
	w.eventHandlers[ctrlConf.Key()] = append(w.eventHandlers[ctrlConf.Key()], eh)

	w.logger = logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher)

	w.logger.Debugf("initialized")
	return nil
}

func (w *K8sResourceWatcher) Run(ctx context.Context) error {
	for k, config := range w.configs {
		configs, err := getConfigFromSecret(ctx, config.Clusters)
		if err != nil {
			return err
		}
		for _, kubeConfig := range configs {
			go func(kube *rest.Config, c *types.Config, handlers []eventhandler.EventHandler) {
				resourceController := controller.Setup(ctx, kube, *c, handlers)
				resourceController.Run(ctx.Done())
			}(kubeConfig, config, w.eventHandlers[k])
		}
	}
	return nil
}

func getConfigFromSecret(ctx context.Context, clusters []string) ([]*rest.Config, error) {
	configs := make([]*rest.Config, 0)
	config := ctrl.GetConfigOrDie()
	cli, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		req, err := labels.NewRequirement(clusterLabel, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		secrets := &corev1.SecretList{}
		if err := cli.List(ctx, secrets, client.MatchingLabelsSelector{labels.NewSelector().Add(*req)}); err != nil {
			return nil, err
		}
		for _, secret := range secrets.Items {
			configs = append(configs, generateRestConfig(&secret))
		}
		configs = append(configs, config)
		return configs, nil
	}
	for _, cluster := range clusters {
		if cluster == defaultCluster {
			configs = append(configs, config)
			continue
		}
		secret := &corev1.Secret{}
		if err := cli.Get(ctx, client.ObjectKey{Name: cluster, Namespace: "vela-system"}, secret); err != nil {
			return nil, err
		}
		configs = append(configs, generateRestConfig(secret))
	}
	return configs, nil
}

func generateRestConfig(secret *corev1.Secret) *rest.Config {
	c := &rest.Config{
		Host: string(secret.Data[clusterEndpoint]),
		TLSClientConfig: rest.TLSClientConfig{
			KeyData:  secret.Data[clusterCertKey],
			CertData: secret.Data[clusterCertData],
		},
	}
	if ca, ok := secret.Data[clusterCAData]; ok {
		c.TLSClientConfig.CAData = ca
	} else {
		c.Insecure = true
	}
	return c
}

func (w *K8sResourceWatcher) Type() string {
	return v1alpha1.SourceTypeResourceWatcher
}
