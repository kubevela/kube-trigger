/*
Copyright 2023 The KubeVela Authors.

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

package workqueue

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

func TestIndexerDelayingQueue_Version(t *testing.T) {
	q := NewIndexerDelayingQueue("test", metaNamespaceKeyFunc)

	var p1 = &podWrap{&corev1.Pod{}}
	p1.ObjectMeta.Name = "ss"
	p1.ObjectMeta.Namespace = "abc"
	p1.ObjectMeta.ResourceVersion = "111"
	var p2 = &podWrap{p1.DeepCopy()}
	p2.ObjectMeta.ResourceVersion = "112"

	q.Add(p1)
	func() {
		item, _ := q.Get()
		// ... handle occur error
		defer q.Done(item)
		q.AddAfter(item, time.Second)
	}()
	q.Add(p2)
	item, _ := q.Get()
	if q.Len() != 1 {
		t.Errorf("expect queue len: 1, but %d", q.Len())
		return
	}
	if item.(*podWrap).ObjectMeta.ResourceVersion != "112" {
		t.Errorf("expect the resource version : 112, but %s", item.(*podWrap).ObjectMeta.ResourceVersion)
		return
	}
	q.Done(item)
	if q.Len() != 0 {
		t.Errorf("expect queue len: 0, but %d", q.Len())
		return
	}
}

func TestIndexerDelayingQueue_Parallel(t *testing.T) {
	q := NewIndexerDelayingQueue("test", metaNamespaceKeyFunc)

	var p1 = &podWrap{&corev1.Pod{}}
	p1.ObjectMeta.Name = "ss"
	p1.ObjectMeta.Namespace = "abc"
	p1.ObjectMeta.ResourceVersion = "111"
	var p2 = &podWrap{p1.DeepCopy()}
	p2.ObjectMeta.ResourceVersion = "112"

	q.Add(p1)
	first := make(chan struct{})
	firstFinished := false
	go func() {
		item, _ := q.Get()
		close(first)
		time.Sleep(time.Second)
		firstFinished = true
		// ... handle occur error
		defer q.Done(item)
		q.AddAfter(item, time.Second)
	}()
	<-first
	q.Add(p2)
	item, _ := q.Get()
	if !firstFinished {
		t.Errorf("expect don't process the same key at the same time")
		return
	}
	if item.(*podWrap).ObjectMeta.ResourceVersion != "112" {
		t.Errorf("expect resource version: \"112\", but %s", item.(*podWrap).ObjectMeta.ResourceVersion)
		return
	}
	q.AddAfter(item, time.Second)
	q.Done(item)

	item, _ = q.Get()
	if item.(*podWrap).ObjectMeta.ResourceVersion != "112" {
		t.Errorf("expect resource version: \"112\", but %s", item.(*podWrap).ObjectMeta.ResourceVersion)
		return
	}
	q.Done(item)

	if q.Len() != 0 {
		t.Errorf("expect queue is empty,but %d", q.Len())
		return
	}
}

func TestIndexerDelayingQueue_ShutDown(t *testing.T) {
	q := NewIndexerDelayingQueue("test", func(obj interface{}) (string, error) {
		return fmt.Sprint(obj), nil
	})
	q.Add("abc")
	q.ShutDown()

	item, shutdown := q.Get()

	if shutdown {
		t.Errorf("expect queue open, but shutdown")
		return
	}

	if item != "abc" {
		t.Errorf("expect item: \"abc\", but %v", item)
		return
	}

	item, shutdown = q.Get()

	if !shutdown {
		t.Errorf("expect queue shutdown, but open")
		return
	}

	q.Add("xxx")
	item, shutdown = q.Get()
	if !shutdown {
		t.Errorf("expect queue shutdown, but open")
		return
	}

	if item != nil {
		t.Errorf("expect item: nil, but %v", item)
		return
	}
}

type podWrap struct {
	*corev1.Pod
}

func (w *podWrap) LessOrEqual(item interface{}) bool {
	return w.GetResourceVersion() <= item.(*podWrap).GetResourceVersion()
}

func TestBufferCap(t *testing.T) {
	q := NewIndexerDelayingQueue("test", func(obj interface{}) (string, error) {
		return fmt.Sprint(obj), nil
	})
	for i := 1; i <= queueItemCap; i++ {
		q.Add(i)
	}
	start := time.Now()
	go func() {
		time.Sleep(time.Second)
		q.ShutDown()
	}()
	q.AddAfter(1000, 0)
	if time.Now().Sub(start).Seconds() >= 1 {
		t.Error("block AddAfter when queue overhead")
	}
	q.Add(1000)
	if time.Now().Sub(start).Seconds() < 1 {
		t.Error("can't block Add when queue overhead")
	}

	q.ShutDown()
	q.Add(1000)
}

func TestIndexerQueueLen(t *testing.T) {
	q := NewIndexerDelayingQueue("test", func(obj interface{}) (string, error) {
		return fmt.Sprint(obj), nil
	})

	q.AddAfter(1, time.Second)
	q.AddAfter(1, time.Second*2)
	q.Add(1)
	if q.Len() != 1 {
		t.Errorf("q.Len() should be 1, but: %d", q.Len())
	}
	item, _ := q.Get()
	if item != 1 {
		t.Errorf("q.Get() should be 1, but: %d", item)
	}
	q.Done(1)
	if q.Len() != 0 {
		t.Errorf("q.Len() should be 0, but: %d", q.Len())
	}
}

func metaNamespaceKeyFunc(obj interface{}) (string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return "", fmt.Errorf("object has no meta: %v", err)
	}
	if len(meta.GetNamespace()) > 0 {
		return meta.GetNamespace() + "/" + meta.GetName(), nil
	}
	return meta.GetName(), nil
}
