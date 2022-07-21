package controller


import (
"context"
"fmt"
"os"
"os/signal"
"strings"
"syscall"
"time"

"k8s.io/apimachinery/pkg/runtime/schema"
"k8s.io/client-go/dynamic"
"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

ctrl "sigs.k8s.io/controller-runtime"

"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

"github.com/bitnami-labs/kubewatch/config"
"github.com/bitnami-labs/kubewatch/pkg/event"
"github.com/bitnami-labs/kubewatch/pkg/handlers"
"github.com/bitnami-labs/kubewatch/pkg/utils"
"github.com/sirupsen/logrus"

apps_v1 "k8s.io/api/apps/v1"
batch_v1 "k8s.io/api/batch/v1"
api_v1 "k8s.io/api/core/v1"
ext_v1beta1 "k8s.io/api/extensions/v1beta1"
rbac_v1beta1 "k8s.io/api/rbac/v1beta1"
meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/apimachinery/pkg/runtime"
utilruntime "k8s.io/apimachinery/pkg/util/runtime"
"k8s.io/apimachinery/pkg/util/wait"
"k8s.io/apimachinery/pkg/watch"
"k8s.io/client-go/kubernetes"
"k8s.io/client-go/rest"
"k8s.io/client-go/tools/cache"
"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

var serverStartTime time.Time

// Event indicate the informerEvent
type Event struct {
	key          string
	eventType    string
	namespace    string
	resourceType string
}

// Controller object
type Controller struct {
	logger       *logrus.Entry
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	eventHandler handlers.Handler
}

// Start prepares watchers and run their controllers, then waits for process termination signals
func Start(conf *config.Config, eventHandler handlers.Handler) {
	var kubeClient kubernetes.Interface

	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}
	ctx := context.Background()
	mapper, err := apiutil.NewDiscoveryRESTMapper(ctrl.GetConfigOrDie())
	if err != nil {
		logrus.Fatal(err)
	}
	if conf.CRD != nil {
		gv, err := schema.ParseGroupVersion(conf.CRD.APIVersion)
		if err != nil {
			logrus.Fatal(err)
		}
		gvk := gv.WithKind(conf.CRD.Kind)

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gv.Version)
		if err != nil {
			logrus.Fatal(err)
		}
		dynamicClient, err := dynamic.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			logrus.Fatal(err)
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

		c := newResourceController(kubeClient, eventHandler, informer, conf.CRD.Kind)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}


	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func newResourceController(client kubernetes.Interface, eventHandler handlers.Handler, informer cache.SharedIndexInformer, resourceType string) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent Event
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "create"
			newEvent.resourceType = resourceType
			logrus.WithField("pkg", "kubewatch-"+resourceType).Infof("Processing add to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.eventType = "update"
			newEvent.resourceType = resourceType
			logrus.WithField("pkg", "kubewatch-"+resourceType).Infof("Processing update to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			newEvent.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.eventType = "delete"
			newEvent.resourceType = resourceType
			newEvent.namespace = utils.GetObjectMetaData(obj).Namespace
			logrus.WithField("pkg", "kubewatch-"+resourceType).Infof("Processing delete to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		logger:       logrus.WithField("pkg", "kubewatch-"+resourceType),
		clientset:    client,
		informer:     informer,
		queue:        queue,
		eventHandler: eventHandler,
	}
}

// Run starts the kubewatch controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("Starting kubewatch controller")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	c.logger.Info("Kubewatch controller synced and ready")

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
	err := c.processItem(newEvent.(Event))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		c.logger.Errorf("Error processing %s (will retry): %v", newEvent.(Event).key, err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		c.logger.Errorf("Error processing %s (giving up): %v", newEvent.(Event).key, err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}

	return true
}

/* TODOs
- Enhance event creation using client-side cacheing machanisms - pending
- Enhance the processItem to classify events - done
- Send alerts correspoding to events - done
*/

func (c *Controller) processItem(newEvent Event) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(newEvent.key)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", newEvent.key, err)
	}
	// get object's metedata
	objectMeta := utils.GetObjectMetaData(obj)

	// hold status type for default critical alerts
	var status string

	// namespace retrived from event key incase namespace value is empty
	if newEvent.namespace == "" && strings.Contains(newEvent.key, "/") {
		substring := strings.Split(newEvent.key, "/")
		newEvent.namespace = substring[0]
		newEvent.key = substring[1]
	}

	// process events based on its type
	switch newEvent.eventType {
	case "create":
		// compare CreationTimestamp and serverStartTime and alert only on latest events
		// Could be Replaced by using Delta or DeltaFIFO
		if objectMeta.CreationTimestamp.Sub(serverStartTime).Seconds() > 0 {
			switch newEvent.resourceType {
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
				Name:      objectMeta.Name,
				Namespace: newEvent.namespace,
				Kind:      newEvent.resourceType,
				Status:    status,
				Reason:    "Created",
			}
			c.eventHandler.Handle(kbEvent)
			return nil
		}
	case "update":
		/* TODOs
		- enahace update event processing in such a way that, it send alerts about what got changed.
		*/
		switch newEvent.resourceType {
		case "Backoff":
			status = "Danger"
		default:
			status = "Warning"
		}
		kbEvent := event.Event{
			Name:      newEvent.key,
			Namespace: newEvent.namespace,
			Kind:      newEvent.resourceType,
			Status:    status,
			Reason:    "Updated",
		}
		c.eventHandler.Handle(kbEvent)
		return nil
	case "delete":
		kbEvent := event.Event{
			Name:      newEvent.key,
			Namespace: newEvent.namespace,
			Kind:      newEvent.resourceType,
			Status:    "Danger",
			Reason:    "Deleted",
		}
		c.eventHandler.Handle(kbEvent)
		return nil
	}
	return nil
}
