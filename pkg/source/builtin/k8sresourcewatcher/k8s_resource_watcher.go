package k8sresourcewatcher

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/config"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/controller"
	krwtypes "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/sirupsen/logrus"
)

type K8sResourceWatcher struct {
	resourceController *controller.Controller
	logger             *logrus.Entry
}

func (w *K8sResourceWatcher) New() sourcetypes.Source {
	return &K8sResourceWatcher{}
}

func (w *K8sResourceWatcher) Init(properties cue.Value,
	filters []filtertypes.FilterMeta,
	filterRegistry *filterregistry.Registry,
) error {
	var err error

	ctrlConf := &config.Config{}
	err = ctrlConf.Parse(properties)
	if err != nil {
		return err
	}

	w.resourceController = controller.Setup(
		*ctrlConf,
		filters,
		filterRegistry,
	)

	w.logger = logrus.WithField("source", krwtypes.TypeName)

	w.logger.Debugf("initialized")
	return nil
}

func (w *K8sResourceWatcher) AddEventHandler(eh sourcetypes.EventHandler) {
	w.resourceController.AddEventHandler(eh)
}

func (w *K8sResourceWatcher) Run(ctx context.Context) error {
	if w.resourceController == nil {
		return fmt.Errorf("controller has not been setup")
	}
	w.logger.Debugf("running")
	w.resourceController.Run(ctx.Done())
	return nil
}

func (w *K8sResourceWatcher) Type() string {
	return krwtypes.TypeName
}
