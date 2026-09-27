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
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

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
			event := item.(types.InformerEvent)
			a.Equal(types.EventTypeDelete, event.Type)
			a.Equal(tc.queued, event.EventObj)
		})
	}
}
