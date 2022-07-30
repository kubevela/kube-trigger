package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/config"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/event"
	krwtypes "github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/utils"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/kubevela/kube-trigger/pkg/workqueue"
	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const maxRetries = 5

var serverStartTime time.Time

// Controller object
type Controller struct {
	logger    *logrus.Entry
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer

	eventHandlers  []types.EventHandler
	sourceConf     config.Config
	filterRegistry *filterregistry.Registry
	filters        []filtertypes.FilterMeta
	listenEvents   map[string]bool
	controllerType string
}

func init() {
	v1beta1.AddToScheme(scheme.Scheme)
}

// Start prepares watchers and run their controllers, then waits for process termination signals
func Setup(ctrlConf config.Config, filters []filtertypes.FilterMeta, filterRegistry *filterregistry.Registry) *Controller {
	conf := ctrl.GetConfigOrDie()
	ctx := context.Background()
	mapper, err := apiutil.NewDiscoveryRESTMapper(conf)
	if err != nil {
		logrus.WithField("source", krwtypes.TypeName).Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(conf)
	if err != nil {
		logrus.WithField("source", krwtypes.TypeName).Fatalf("Can not create kubernetes client: %v", err)
	}
	//cc, err := client.New(conf, client.Options{Scheme: scheme.Scheme})
	//if err != nil {
	//		logrus.WithField("source", krwtypes.TypeName).Fatalf("Can not create kubernetes client: %v", err)
	//}
	gv, err := schema.ParseGroupVersion(ctrlConf.APIVersion)
	if err != nil {
		logrus.WithField("source", krwtypes.TypeName).Fatal(err)
	}
	gvk := gv.WithKind(ctrlConf.Kind)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gv.Version)
	if err != nil {
		logrus.WithField("source", krwtypes.TypeName).Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		logrus.WithField("source", krwtypes.TypeName).Fatal(err)
	}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return dynamicClient.Resource(mapping.Resource).List(ctx, options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(mapping.Resource).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(
		kubeClient,
		informer,
		ctrlConf.Kind,
	)
	c.sourceConf = ctrlConf
	c.filters = filters
	c.filterRegistry = filterRegistry

	listenEvents := make(map[string]bool)
	for _, e := range c.sourceConf.Events {
		listenEvents[e] = true
	}
	c.listenEvents = listenEvents

	c.controllerType = krwtypes.TypeName

	//stopCh := make(chan struct{})
	//defer close(stopCh)
	//
	//go c.Run(stopCh)
	//
	//sigterm := make(chan os.Signal, 1)
	//signal.Notify(sigterm, syscall.SIGTERM)
	//signal.Notify(sigterm, syscall.SIGINT)
	//<-sigterm

	return c
}

func newResourceController(
	client kubernetes.Interface,
	informer cache.SharedIndexInformer,
	resourceType string,
) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent event.InformerEvent
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.Key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.EventType = "create"
			newEvent.ResourceType = resourceType
			logrus.WithField("source", krwtypes.TypeName).Tracef("received add event: %v %s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newEvent.Key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.EventType = "update"
			newEvent.ResourceType = resourceType
			logrus.WithField("source", krwtypes.TypeName).Tracef("received update event: %v %s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			newEvent.Key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.EventType = "delete"
			newEvent.ResourceType = resourceType
			newEvent.Namespace = utils.GetObjectMetaData(obj).GetNamespace()
			logrus.WithField("source", krwtypes.TypeName).Tracef("received delete event: %v d%s", resourceType, newEvent.Key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		logger:    logrus.WithField("source", krwtypes.TypeName),
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
	err := c.processItem(newEvent.(event.InformerEvent))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		c.logger.Errorf("error processing %s (will retry): %v", newEvent.(event.InformerEvent).Key, err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		c.logger.Errorf("error processing %s (giving up): %v", newEvent.(event.InformerEvent).Key, err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(newEvent event.InformerEvent) error {
	// Ignore exists, we want deleting event as well
	obj, _, err := c.informer.GetIndexer().GetByKey(newEvent.Key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", newEvent.Key, err)
	}
	// get object's metadata
	objectMeta := utils.GetObjectMetaData(obj)
	c.logger.Tracef("processong object %v", objectMeta)

	// hold status type for default critical alerts
	var status string

	// namespace retrieved from event key in case namespace value is empty
	if newEvent.Namespace == "" && strings.Contains(newEvent.Key, "/") {
		substring := strings.Split(newEvent.Key, "/")
		newEvent.Namespace = substring[0]
		newEvent.Key = substring[1]
	}

	if c.sourceConf.Namespace != "" && c.sourceConf.Namespace != newEvent.Namespace {
		c.logger.Debugf("object %s filtered out because of different namespaces: %s!=%s", newEvent.Key, c.sourceConf.Namespace, newEvent.Namespace)
		return nil
	}

	if len(c.listenEvents) > 0 && !c.listenEvents[newEvent.EventType] {
		c.logger.Debugf("object filtered out because of not specified event type: %s", newEvent.EventType)
		return nil
	}

	// process events based on its type
	switch newEvent.EventType {
	case "create":
		// compare CreationTimestamp and serverStartTime and alert only on latest events
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
			kbEvent := event.Event{
				Name:      objectMeta.GetName(),
				Namespace: newEvent.Namespace,
				Kind:      newEvent.ResourceType,
				Info:      status,
				Obj:       objectMeta,
			}
			c.logger.Debugf("add create event: %s", kbEvent.Message())
			c.doFilteringAndCallHandlers(kbEvent)

			return nil
		}
	case "update":

		switch newEvent.ResourceType {
		case "Backoff":
			status = "Danger"
		default:
			status = "Warning"
		}
		kbEvent := event.Event{
			Name:      newEvent.Key,
			Namespace: newEvent.Namespace,
			Kind:      newEvent.ResourceType,
			Info:      status,
			Obj:       objectMeta,
		}
		c.logger.Debugf("add update event: %s", kbEvent.Message())
		c.doFilteringAndCallHandlers(kbEvent)
		return nil
	case "delete":
		kbEvent := event.Event{
			Name:      newEvent.Key,
			Namespace: newEvent.Namespace,
			Kind:      newEvent.ResourceType,
			Info:      "Deleted",
			Obj:       objectMeta,
		}
		c.logger.Debugf("add create event: %s", kbEvent.Message())
		c.doFilteringAndCallHandlers(kbEvent)
		return nil
	}
	return nil
}

func (c *Controller) doFilteringAndCallHandlers(event event.Event) {
	c.logger.Debugf("applying filters to event name %s", event.Name)
	// apply filters
	for _, f := range c.filters {
		fInstance, err := c.filterRegistry.CreateOrGetInstance(f)
		if err != nil {
			return
		}
		kept, err := fInstance.ApplyToObject(event.Obj)
		if err != nil || !kept {
			c.logger.Debugf("event name %s filtered out by filter %s", event.Message(), fInstance.Type())
			return
		}
	}

	c.logger.Debugf("calling event handlers by event name %s", event.Name)
	for _, h := range c.eventHandlers {
		h(c.controllerType, event)
	}
}

func (c *Controller) AddEventHandler(h types.EventHandler) {
	c.eventHandlers = append(c.eventHandlers, h)
	c.logger.Debugf("add event handler %d", len(c.eventHandlers))
}
