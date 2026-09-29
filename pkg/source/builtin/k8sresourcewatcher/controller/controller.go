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
	"slices"
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
	"k8s.io/client-go/tools/cache"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/utils"
	"github.com/kubevela/kube-trigger/pkg/workqueue"
)

const maxRetries = 5

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
	// startTime is when Run started; creates of objects older than it are
	// skipped, as the initial list reports them too.
	startTime time.Time
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
		&cache.ListWatch{
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
		cache.Indexers{},
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
	cluster, _ := multicluster.ClusterFrom(ctx)
	//nolint:errcheck // no need to check err here
	informer.AddEventHandler(resourceEventHandler(logger, queue, kind, cluster))

	return &Controller{
		logger:   logger,
		informer: informer,
		queue:    queue,
		cluster:  cluster,
	}
}

// ForEvents calls eh only for the event types its trigger asked for; no events
// means all of them. Triggers watching the same resources share one watcher,
// which listens for every event any of them asked for.
func ForEvents(events []types.EventType, eh eventhandler.EventHandler) eventhandler.EventHandler {
	if len(events) == 0 {
		return eh
	}
	events = slices.Clone(events)
	return func(sourceType string, event interface{}, data interface{}) error {
		if !slices.Contains(events, event.(types.Event).Type) {
			return nil
		}
		return eh(sourceType, event, data)
	}
}

// resourceEventHandler queues each informer event for the worker.
func resourceEventHandler(logger *logrus.Entry, queue workqueue.RateLimitingInterface, kind string, cluster string) cache.ResourceEventHandlerFuncs {
	enqueue := func(typ types.EventType, obj interface{}) {
		meta, ok := utils.GetObjectMetaData(obj)
		if !ok {
			logger.Warnf("skipping %s event for %v: no object in %T", typ, kind, obj)
			return
		}
		logger.Tracef("received %s event: %v %s/%s", typ, kind, meta.GetName(), meta.GetNamespace())
		queue.Add(types.InformerEvent{
			Event:    types.Event{Type: typ, Cluster: cluster},
			EventObj: meta,
		})
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			enqueue(types.EventTypeCreate, obj)
		},
		UpdateFunc: func(old, new interface{}) {
			// A relist reports every cached object as an update; an unchanged
			// resourceVersion means nothing was written.
			oldMeta, oldOK := utils.GetObjectMetaData(old)
			newMeta, newOK := utils.GetObjectMetaData(new)
			if oldOK && newOK && oldMeta.GetResourceVersion() != "" && oldMeta.GetResourceVersion() == newMeta.GetResourceVersion() {
				return
			}
			enqueue(types.EventTypeUpdate, new)
		},
		DeleteFunc: func(obj interface{}) {
			enqueue(types.EventTypeDelete, obj)
		},
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
	c.startTime = time.Now().Local()

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

	meta := newEvent.(types.InformerEvent).EventObj
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
	objectMeta := newEvent.EventObj
	// Fetching (create,update,delete) event Obj of k8s
	c.logger.Debugf("Fetching obj (%+v) with newEvent(%s/%s) and eventType=%s from event", newEvent.EventObj, objectMeta.GetName(), objectMeta.GetNamespace(), newEvent.Type)

	if len(c.listenEvents) > 0 && !c.listenEvents[newEvent.Type] {
		c.logger.Debugf("object filtered out because of not specified event type: %s", newEvent.Type)
		return nil
	}

	// Process events based on its type
	switch newEvent.Type {
	case types.EventTypeCreate:
		// Compare CreationTimestamp and startTime and alert only on latest events
		// Could be Replaced by using Delta or DeltaFIFO
		if objectMeta.GetCreationTimestamp().Sub(c.startTime).Seconds() > 0 {
			c.logger.Debugf("add %s event: %s/%s", newEvent.Type, objectMeta.GetName(), objectMeta.GetNamespace())
			c.callEventHandler(objectMeta, newEvent.Event)
			return nil
		}
	default:
		c.logger.Debugf("add %s event: %s/%s", newEvent.Type, objectMeta.GetName(), objectMeta.GetNamespace())
		c.callEventHandler(objectMeta, newEvent.Event)
	}
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
