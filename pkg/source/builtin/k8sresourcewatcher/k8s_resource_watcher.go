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
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/kubevela/pkg/multicluster"
	"github.com/kubevela/pkg/util/singleton"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/controller"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
)

func init() {
	cfg := singleton.KubeConfig.Get()
	cfg.Wrap(multicluster.NewTransportWrapper())
	singleton.KubeConfig.Set(cfg)
	singleton.ReloadClients()
}

var (
	// MultiClusterConfigType .
	MultiClusterConfigType string
)

const (
	// TypeClusterGateway .
	TypeClusterGateway string = "cluster-gateway"
	// TypeClusterGatewaySecret .
	TypeClusterGatewaySecret string = "cluster-gateway-secret"

	clusterLabel    string = "cluster.core.oam.dev/cluster-credential-type"
	defaultCluster  string = "local"
	clusterCertKey  string = "tls.key"
	clusterCertData string = "tls.crt"
	clusterCAData   string = "ca.crt"
	clusterEndpoint string = "endpoint"
)

// K8sResourceWatcher watches k8s resources.
type K8sResourceWatcher struct {
	configs       map[string]*types.Config
	eventHandlers map[string][]eventhandler.EventHandler
	logger        *logrus.Entry
}

var _ sourcetypes.Source = &K8sResourceWatcher{}

// New .
func (w *K8sResourceWatcher) New() sourcetypes.Source {
	return &K8sResourceWatcher{
		configs:       make(map[string]*types.Config),
		eventHandlers: make(map[string][]eventhandler.EventHandler),
	}
}

// Parse .
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

// Init .
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

	w.logger = logrus.WithField("source-type", v1alpha1.SourceTypeResourceWatcher)

	w.logger.Debugf("initialized")
	return nil
}

// Run .
func (w *K8sResourceWatcher) Run(ctx context.Context) error {
	clusterGetter, err := NewMultiClustersGetter(MultiClusterConfigType)
	if err != nil {
		return err
	}
	for k, config := range w.configs {
		if len(config.Clusters) == 0 {
			config.Clusters = []string{defaultCluster}
		}
		for _, cluster := range config.Clusters {
			cli, mapper, err := clusterGetter.GetDynamicClientAndMapper(ctx, cluster)
			if err != nil {
				return err
			}
			multiCtx := multicluster.WithCluster(ctx, cluster)
			go func(multiCtx context.Context, cli dynamic.Interface, mapper meta.RESTMapper, c *types.Config, handlers []eventhandler.EventHandler) {
				resourceController := controller.Setup(multiCtx, cli, mapper, *c, handlers)
				resourceController.Run(multiCtx.Done())
			}(multiCtx, cli, mapper, config, w.eventHandlers[k])
		}
	}
	return nil
}

// Type .
func (w *K8sResourceWatcher) Type() string {
	return v1alpha1.SourceTypeResourceWatcher
}

// Singleton .
func (w *K8sResourceWatcher) Singleton() bool {
	return true
}

// MultiClustersGetter .
type MultiClustersGetter interface {
	GetDynamicClientAndMapper(ctx context.Context, cluster string) (dynamic.Interface, meta.RESTMapper, error)
}

// NewMultiClustersGetter new a MultiClustersGetter
func NewMultiClustersGetter(typ string) (MultiClustersGetter, error) {
	config := ctrl.GetConfigOrDie()
	cli, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	switch typ {
	case TypeClusterGateway:
		return &clusterGatewayGetter{}, nil
	case TypeClusterGatewaySecret:
		return &clusterGatewaySecretGetter{cli: cli, config: config}, nil
	default:
		return nil, fmt.Errorf("unknown multi-cluster getter type %s", typ)
	}
}

type clusterGatewayGetter struct{}

func (c *clusterGatewayGetter) GetDynamicClientAndMapper(_ context.Context, _ string) (dynamic.Interface, meta.RESTMapper, error) {
	return singleton.DynamicClient.Get(), singleton.RESTMapper.Get(), nil
}

type clusterGatewaySecretGetter struct {
	cli    client.Client
	config *rest.Config
}

func (c *clusterGatewaySecretGetter) GetDynamicClientAndMapper(ctx context.Context, cluster string) (dynamic.Interface, meta.RESTMapper, error) {
	if cluster == defaultCluster {
		return c.getDynamicClientAndMapperFromConfig(ctx, c.config)
	}
	config, err := c.getRestConfigFromSecret(ctx, cluster)
	if err != nil {
		return nil, nil, err
	}

	return c.getDynamicClientAndMapperFromConfig(ctx, config)
}

func (c *clusterGatewaySecretGetter) getDynamicClientAndMapperFromConfig(_ context.Context, config *rest.Config) (dynamic.Interface, meta.RESTMapper, error) {
	cli, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, nil, err
	}
	return cli, mapper, nil
}

func (c *clusterGatewaySecretGetter) getRestConfigFromSecret(ctx context.Context, cluster string) (*rest.Config, error) {
	secret := &corev1.Secret{}
	if err := c.cli.Get(ctx, client.ObjectKey{Name: cluster, Namespace: "vela-system"}, secret); err != nil {
		return nil, err
	}
	// currently we only type X509
	// TODO: support other types
	conf := &rest.Config{
		Host: string(secret.Data[clusterEndpoint]),
		TLSClientConfig: rest.TLSClientConfig{
			KeyData:  secret.Data[clusterCertKey],
			CertData: secret.Data[clusterCertData],
		},
	}
	if ca, ok := secret.Data[clusterCAData]; ok {
		conf.TLSClientConfig.CAData = ca
	} else {
		conf.Insecure = true
	}
	return conf, nil
}
