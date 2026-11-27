/*
Copyright 2026 The KubeVela Authors.

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
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/types"
	"github.com/kubevela/kube-trigger/pkg/workqueue"
)

func TestDeleteHandlesTombstones(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("ConfigMap")
	obj.SetName("watched")
	obj.SetNamespace("default")

	testcases := map[string]struct {
		deleted interface{}
		queued  interface{}
	}{
		"plain delete queues the object": {
			deleted: obj,
			queued:  obj,
		},
		// The informer hands over a tombstone when it missed the delete and only
		// saw the object gone on a relist.
		"tombstone queues its last known object": {
			deleted: cache.DeletedFinalStateUnknown{Key: "default/watched", Obj: obj},
			queued:  obj,
		},
		"tombstone without an object is skipped": {
			deleted: cache.DeletedFinalStateUnknown{Key: "default/watched", Obj: nil},
			queued:  nil,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			defer queue.ShutDown()
			handler := resourceEventHandler(logrus.NewEntry(logrus.New()), queue, "ConfigMap", "local")

			a.NotPanics(func() { handler.OnDelete(tc.deleted) })

			if tc.queued == nil {
				a.Equal(0, queue.Len())
				return
			}
			if !a.Equal(1, queue.Len()) {
				return
			}
			item, _ := queue.Get()
			event := item.(*types.InformerEvent)
			a.Equal(types.EventTypeDelete, event.Type)
			a.Equal(tc.queued, event.EventObj)
		})
	}
}

func TestForEventsOnlyPassesItsTriggersEvents(t *testing.T) {
	testcases := map[string]struct {
		events []types.EventType
		called []types.EventType
	}{
		"trigger watching create": {
			events: []types.EventType{types.EventTypeCreate},
			called: []types.EventType{types.EventTypeCreate},
		},
		"trigger watching update and delete": {
			events: []types.EventType{types.EventTypeUpdate, types.EventTypeDelete},
			called: []types.EventType{types.EventTypeUpdate, types.EventTypeDelete},
		},
		"trigger with no events gets all": {
			events: nil,
			called: []types.EventType{types.EventTypeCreate, types.EventTypeUpdate, types.EventTypeDelete},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			var called []types.EventType
			eh := ForEvents(tc.events, func(p eventhandler.Payload) error {
				called = append(called, p.Event.(types.Event).Type)
				return nil
			})
			for _, typ := range []types.EventType{types.EventTypeCreate, types.EventTypeUpdate, types.EventTypeDelete} {
				assert.NoError(t, eh(eventhandler.Payload{SourceType: "resource-watcher", Event: types.Event{Type: typ}}))
			}
			assert.Equal(t, tc.called, called)
		})
	}
}

func TestWatcherStartingLaterKeepsEarlierWatchersCreates(t *testing.T) {
	a := assert.New(t)
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}, meta.RESTScopeNamespace)
	scheme := runtime.NewScheme()
	a.NoError(corev1.AddToScheme(scheme))
	cli := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: "ConfigMapList"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled int
	start := func(eh eventhandler.EventHandler) *Controller {
		c := Setup(ctx, cli, mapper, types.Config{APIVersion: "v1", Kind: "ConfigMap"}, []eventhandler.EventHandler{eh})
		go c.Run(ctx.Done())
		a.NoError(wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, func(_ context.Context) (bool, error) {
			return c.HasSynced(), nil
		}))
		return c
	}
	first := start(func(_ eventhandler.Payload) error {
		handled++
		return nil
	})

	// Created after the first watcher started, at the second precision of
	// creationTimestamp, and before the second watcher starts.
	time.Sleep(1100 * time.Millisecond)
	obj := &unstructured.Unstructured{}
	obj.SetName("created")
	obj.SetCreationTimestamp(metav1.Now())
	time.Sleep(1100 * time.Millisecond)
	start(func(_ eventhandler.Payload) error { return nil })

	a.NoError(first.processItem(&types.InformerEvent{Event: types.Event{Type: types.EventTypeCreate}, EventObj: obj}))
	a.Equal(1, handled)
}

func TestUpdateWithoutAChangeIsSkipped(t *testing.T) {
	configMap := func(resourceVersion string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{}
		obj.SetName("watched")
		obj.SetResourceVersion(resourceVersion)
		return obj
	}

	testcases := map[string]struct {
		old, new *unstructured.Unstructured
		queued   int
	}{
		"a write changes the resourceVersion": {
			old:    configMap("1"),
			new:    configMap("2"),
			queued: 1,
		},
		// A relist reports every cached object as an update, changed or not.
		"a relist of an unchanged object": {
			old:    configMap("1"),
			new:    configMap("1"),
			queued: 0,
		},
		// Only a fake or hand-built object lacks one; there is nothing to compare.
		"no resourceVersion on either": {
			old:    configMap(""),
			new:    configMap(""),
			queued: 1,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			defer queue.ShutDown()
			handler := resourceEventHandler(logrus.NewEntry(logrus.New()), queue, "ConfigMap", "local")

			handler.OnUpdate(tc.old, tc.new)
			assert.Equal(t, tc.queued, queue.Len())
		})
	}
}

func TestUpdateCarriesWhatChanged(t *testing.T) {
	a := assert.New(t)
	configMap := func(resourceVersion, value string) *unstructured.Unstructured {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{"data": map[string]interface{}{"key": value}}}
		obj.SetName("watched")
		obj.SetResourceVersion(resourceVersion)
		return obj
	}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	handler := resourceEventHandler(logrus.NewEntry(logrus.New()), queue, "ConfigMap", "local")

	handler.OnAdd(configMap("1", "a"), false)
	handler.OnUpdate(configMap("1", "a"), configMap("2", "b"))
	handler.OnDelete(configMap("2", "b"))

	var changed []map[string]interface{}
	for queue.Len() > 0 {
		item, _ := queue.Get()
		changed = append(changed, item.(*types.InformerEvent).Changed)
		queue.Done(item)
	}
	a.Equal([]map[string]interface{}{
		nil,
		{"metadata": map[string]interface{}{"resourceVersion": "2"}, "data": map[string]interface{}{"key": "b"}},
		nil,
	}, changed)
}

type handled struct {
	eventType types.EventType
	version   string
	changed   interface{}
}

// version reads the label the test bumps on each write.
func version(obj interface{}) string {
	return obj.(metav1.Object).GetLabels()["version"]
}

func TestHandlersReceiveChangedOnUpdate(t *testing.T) {
	a := assert.New(t)
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}, meta.RESTScopeNamespace)
	scheme := runtime.NewScheme()
	a.NoError(corev1.AddToScheme(scheme))
	cli := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: "ConfigMapList"})

	var mu sync.Mutex
	var got []handled
	eh := eventhandler.EventHandler(func(p eventhandler.Payload) error {
		mu.Lock()
		defer mu.Unlock()
		var changed interface{}
		if p.Changed != nil {
			changed = p.Changed["metadata"].(map[string]interface{})["labels"]
		}
		got = append(got, handled{p.Event.(types.Event).Type, version(p.Data), changed})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := Setup(ctx, cli, mapper, types.Config{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Namespace:  "default",
		Events:     []types.EventType{types.EventTypeCreate, types.EventTypeUpdate, types.EventTypeDelete},
	}, []eventhandler.EventHandler{eh})
	go c.Run(ctx.Done())
	a.NoError(wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, func(_ context.Context) (bool, error) {
		return c.HasSynced(), nil
	}))
	// Create events are only handled for objects created after the watcher
	// started, compared at the second precision of creationTimestamp.
	time.Sleep(time.Second)

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("ConfigMap")
	obj.SetName("watched")
	obj.SetNamespace("default")
	obj.SetCreationTimestamp(metav1.Now())
	obj.SetLabels(map[string]string{"version": "1"})
	res := cli.Resource(gvr).Namespace("default")
	_, err := res.Create(ctx, obj, metav1.CreateOptions{})
	a.NoError(err)
	obj.SetLabels(map[string]string{"version": "2"})
	// The fake client leaves resourceVersion to the caller.
	obj.SetResourceVersion("2")
	_, err = res.Update(ctx, obj, metav1.UpdateOptions{})
	a.NoError(err)
	a.NoError(res.Delete(ctx, "watched", metav1.DeleteOptions{}))

	a.NoError(wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, func(_ context.Context) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == 3, nil
	}))
	a.Equal([]handled{
		{types.EventTypeCreate, "1", nil},
		{types.EventTypeUpdate, "2", map[string]interface{}{"version": "2"}},
		{types.EventTypeDelete, "2", nil},
	}, got)
}
