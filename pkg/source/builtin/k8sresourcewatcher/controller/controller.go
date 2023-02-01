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
	"strings"
	"time"

	"github.com/oam-dev/kubevela-core-api/apis/core.oam.dev/v1beta1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/kubevela/kube-trigger/api/v1alpha1"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/utils"
)

const maxRetries = 5

var serverStartTime time.Time

// Controller object
type Controller struct {
	logger    *logrus.Entry
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer

	eventHandlers  []eventhandler.EventHandler
	sourceConf     types.Config
	listenEvents   map[types.EventType]bool
	controllerType string
}

func init() {
	v1beta1.AddToScheme(scheme.Scheme)
}

// Setup prepares controllers
func Setup(ctrlConf types.Config, eh []eventhandler.EventHandler) *Controller {
	conf := ctrl.GetConfigOrDie()
	ctx := context.Background()
	mapper, err := apiutil.NewDiscoveryRESTMapper(conf)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatalf("Can not create kubernetes client: %v", err)
	}
	gv, err := schema.ParseGroupVersion(ctrlConf.APIVersion)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}
	gvk := gv.WithKind(ctrlConf.Kind)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gv.Version)
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Fatal(err)
	}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if len(ctrlConf.MatchingLabels) > 0 {
					options.LabelSelector = labels.FormatLabels(ctrlConf.MatchingLabels)
				}
				return dynamicClient.Resource(mapping.Resource).Namespace(ctrlConf.Namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if len(ctrlConf.MatchingLabels) > 0 {
					options.LabelSelector = labels.FormatLabels(ctrlConf.MatchingLabels)
				}
				return dynamicClient.Resource(mapping.Resource).Namespace(ctrlConf.Namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		0, // Skip resync
		cache.Indexers{},
	)

	c := newResourceController(
		kubeClient,
		informer,
		ctrlConf.Kind,
	)
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

func newResourceController(
	client kubernetes.Interface,
	informer cache.SharedIndexInformer,
	resourceType string,
) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent types.InformerEvent
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.Key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.Type = types.EventTypeCreate
			newEvent.ResourceType = resourceType
			newEvent.EventObj = obj
			logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Tracef("received add event: %v %s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newEvent.Key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.Type = types.EventTypeUpdate
			newEvent.ResourceType = resourceType
			newEvent.EventObj = new
			logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Tracef("received update event: %v %s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			newEvent.Key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.Type = types.EventTypeDelete
			newEvent.ResourceType = resourceType
			newEvent.EventObj = obj
			newEvent.Namespace = utils.GetObjectMetaData(obj).GetNamespace()
			logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher).Tracef("received delete event: %v %s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		logger:    logrus.WithField("source", v1alpha1.SourceTypeResourceWatcher),
		clientset: client,
		informer:  informer,
		queue:     queue,
	}
}

// Run starts the kube-trigger controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("starting...")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	c.logger.Info("synced and ready")

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

	err := c.processItem(newEvent.(types.InformerEvent))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		c.logger.Errorf("error processing %s (will retry): %v", newEvent.(types.InformerEvent).Key, err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		c.logger.Errorf("error processing %s (giving up): %v", newEvent.(types.InformerEvent).Key, err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(newEvent types.InformerEvent) error {
	// Fetching (create,update,delete) event Obj of k8s
	c.logger.Debugf("Fetching obj  (%+v) with newEvent key=%s and eventType=%s from event", newEvent.EventObj, newEvent.Key, newEvent.Type)

	// Get object's metadata
	objectMeta := utils.GetObjectMetaData(newEvent.EventObj)

	// Hold status type for default critical alerts
	var status string

	// namespace retrieved from event key in case namespace value is empty
	if newEvent.Namespace == "" && strings.Contains(newEvent.Key, "/") {
		substring := strings.Split(newEvent.Key, "/")
		newEvent.Namespace = substring[0]
		newEvent.Key = substring[1]
	}

	if len(c.listenEvents) > 0 && !c.listenEvents[newEvent.Type] {
		c.logger.Debugf("object filtered out because of not specified event type: %s", newEvent.Type)
		return nil
	}

	// Process events based on its type
	switch newEvent.Type {
	case types.EventTypeCreate:
		// Compare CreationTimestamp and serverStartTime and alert only on latest events
		// Could be Replaced by using Delta or DeltaFIFO
		if objectMeta.GetCreationTimestamp().Sub(serverStartTime).Seconds() > 0 {
			switch newEvent.ResourceType {
			case "NodeNotReady":
				status = "Danger"
			case "NodeReady":
				status = "Normal"
			case "NodeRebooted":
				status = "Danger"
			case "Backoff":
				status = "Danger"
			default:
				status = "Normal"
			}
			kbEvent := types.Event{
				Type:      newEvent.Type,
				Name:      objectMeta.GetName(),
				Namespace: newEvent.Namespace,
				Kind:      newEvent.ResourceType,
				Info:      status,
			}
			c.logger.Debugf("add create event: %s", kbEvent.Message())
			c.callEventHandler(objectMeta, kbEvent)
			return nil
		}
	case types.EventTypeUpdate:
		switch newEvent.ResourceType {
		case "Backoff":
			status = "Danger"
		default:
			status = "Warning"
		}
		kbEvent := types.Event{
			Type:      newEvent.Type,
			Name:      newEvent.Key,
			Namespace: newEvent.Namespace,
			Kind:      newEvent.ResourceType,
			Info:      status,
		}
		c.logger.Debugf("add update event: %s", kbEvent.Message())
		c.callEventHandler(objectMeta, kbEvent)
		return nil
	case types.EventTypeDelete:
		kbEvent := types.Event{
			Type:      newEvent.Type,
			Name:      newEvent.Key,
			Namespace: newEvent.Namespace,
			Kind:      newEvent.ResourceType,
			Info:      "Deleted",
		}
		c.logger.Debugf("add delete event: %s", kbEvent.Message())
		c.callEventHandler(objectMeta, kbEvent)
		return nil
	}
	return nil
}

func (c *Controller) callEventHandler(obj metav1.Object, e types.Event) {
	c.logger.Infof("event \"%s\" happened, calling event handlers", e.Message())
	for _, fn := range c.eventHandlers {
		err := fn(c.controllerType, e, obj)
		if err != nil {
			c.logger.Infof("calling event handler failed: %s", err)
		}
	}
}
