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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/kubevela/pkg/multicluster"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	clientcache "k8s.io/client-go/tools/cache"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/cache"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/utils"
	"github.com/kubevela/kube-trigger/pkg/workqueue"
)

const maxRetries = 5

var serverStartTime time.Time

// Controller object
type Controller struct {
	logger   *logrus.Entry
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer

	eventHandlers  []eventhandler.EventHandler
	sourceConf     types.Config
	listenEvents   map[types.EventType]bool
	controllerType string
	cluster        string
}

// Setup prepares controllers
func Setup(ctx context.Context, cli dynamic.Interface, mapper meta.RESTMapper, ctrlConf types.Config, eh []eventhandler.EventHandler) *Controller {
	logger := logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher)
	gv, err := schema.ParseGroupVersion(ctrlConf.APIVersion)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}
	gvk := gv.WithKind(ctrlConf.Kind)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gv.Version)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}

	informer := cache.NewSharedIndexInformer(
		&clientcache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if len(ctrlConf.MatchingLabels) > 0 {
					options.LabelSelector = labels.FormatLabels(ctrlConf.MatchingLabels)
				}
				return cli.Resource(mapping.Resource).Namespace(ctrlConf.Namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if len(ctrlConf.MatchingLabels) > 0 {
					options.LabelSelector = labels.FormatLabels(ctrlConf.MatchingLabels)
				}
				return cli.Resource(mapping.Resource).Namespace(ctrlConf.Namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		0, // Skip resync
		clientcache.Indexers{},
	)

	c := newResourceController(ctx, logger, informer, ctrlConf.Kind)
	// precheck ->
	c.sourceConf = ctrlConf
	c.eventHandlers = eh

	listenEvents := make(map[types.EventType]bool)
	for _, e := range c.sourceConf.Events {
		listenEvents[e] = true
	}
	c.listenEvents = listenEvents

	c.controllerType = v1alpha1.SourceTypeResourceWatcher

	return c
}

func newResourceController(ctx context.Context, logger *logrus.Entry, informer cache.SharedIndexInformer, kind string) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent types.InformerEvent
	var err error
	cluster, _ := multicluster.ClusterFrom(ctx)
	//nolint:errcheck // no need to check err here
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.Event = types.Event{
				Type:    types.EventTypeCreate,
				Cluster: cluster,
			}
			newEvent.EventObj = obj
			meta := utils.GetObjectMetaData(obj)
			logger.Tracef("received add event: %v %s/%s", kind, meta.GetName(), meta.GetNamespace())
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newEvent.Event = types.Event{
				Type:    types.EventTypeUpdate,
				Cluster: cluster,
			}
			newEvent.EventObj = new
			meta := utils.GetObjectMetaData(new)
			logger.Tracef("received update event: %v %s/%s", kind, meta.GetName(), meta.GetNamespace())
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			newEvent.Event = types.Event{
				Type:    types.EventTypeDelete,
				Cluster: cluster,
			}
			newEvent.EventObj = obj
			meta := utils.GetObjectMetaData(obj)
			logger.Tracef("received delete event: %v %s/%s", kind, meta.GetName(), meta.GetNamespace())
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		logger:   logger,
		informer: informer,
		queue:    queue,
		cluster:  cluster,
	}
}

// Run starts the kube-trigger controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	c.logger = c.logger.WithFields(logrus.Fields{
		"apiVersion": c.sourceConf.APIVersion,
		"kind":       c.sourceConf.Kind,
		"cluster":    c.cluster,
	})
	c.logger.Info("starting watch k8s resources...")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	c.logger.Info("resource watcher synced resources and ready for work")
	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *Controller) processNextItem() bool {
	newEvent, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(newEvent)

	meta := utils.GetObjectMetaData(newEvent.(types.InformerEvent).EventObj)
	err := c.processItem(newEvent.(types.InformerEvent))
	//nolint:gocritic // no need to use switch statement here
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		c.logger.Errorf("error processing %s/%s (will retry): %v", meta.GetName(), meta.GetNamespace(), err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		c.logger.Errorf("error processing %s/%s (giving up): %v", meta.GetName(), meta.GetNamespace(), err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(newEvent types.InformerEvent) error {
	// Get object's metadata
	objectMeta := utils.GetObjectMetaData(newEvent.EventObj)
	// Fetching (create,update,delete) event Obj of k8s
	c.logger.Debugf("Fetching obj (%+v) with newEvent(%s/%s) and eventType=%s from event", newEvent.EventObj, objectMeta.GetName(), objectMeta.GetNamespace(), newEvent.Type)

	if len(c.listenEvents) > 0 && !c.listenEvents[newEvent.Type] {
		c.logger.Debugf("object filtered out because of not specified event type: %s", newEvent.Type)
		return nil
	}

	// Process events based on its type
	c.logger.Debugf("add %s event: %s/%s", newEvent.Type, objectMeta.GetName(), objectMeta.GetNamespace())
	c.callEventHandler(objectMeta, newEvent.Event)
	return nil
}

func (c *Controller) callEventHandler(obj metav1.Object, e types.Event) {
	c.logger.Infof("%s event %s/%s/%s happened, calling event handlers", e.Type, e.Cluster, obj.GetNamespace(), obj.GetName())
	for _, fn := range c.eventHandlers {
		err := fn(c.controllerType, e, obj)
		if err != nil {
			c.logger.Infof("calling event handler failed: %s", err)
		}
	}
}
