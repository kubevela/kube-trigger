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

package executor

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

func TestNormalJobs(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              3,
		MaxJobRetries:        0,
		BaseRetryDelay:       10 * time.Millisecond,
		RetryJobAfterFailure: false,
		PerWorkerQPS:         5,
		Timeout:              200 * time.Millisecond,
	}

	e, err := New(c)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 0))

	for i := 1; i <= 3; i++ {
		err = e.AddJob(&sleepingJob{100 * time.Millisecond, fmt.Sprint(i)})
		a.NoError(err)
		a.NoError(waitForAdded(e.queue, i))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	// Wait for all jobs to run
	err = wait.Poll(1*time.Millisecond, 200*time.Millisecond, func() (done bool, err error) {
		return e.claimedIDs() == 3, nil
	})
	a.NoError(err)

	// Wait for all jobs to end
	err = wait.Poll(1*time.Millisecond, 200*time.Millisecond, func() (done bool, err error) {
		return e.claimedIDs() == 0, nil
	})
	a.NoError(err)

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
}

func TestQueueSizeLimits(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	a := assert.New(t)
	c := Config{
		QueueSize:            3,
		Workers:              3,
		MaxJobRetries:        0,
		BaseRetryDelay:       1 * time.Second,
		RetryJobAfterFailure: false,
		PerWorkerQPS:         5,
		Timeout:              5 * time.Second,
	}

	e, err := New(c)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 0))

	for i := 1; i <= 3; i++ {
		err = e.AddJob(&sleepingJob{10 * time.Second, fmt.Sprint(i)})
		a.NoError(err)
		a.NoError(waitForAdded(e.queue, i))
	}

	// Queue full
	err = e.AddJob(&sleepingJob{10 * time.Second, "4"})
	a.Error(err)
	a.NoError(waitForAdded(e.queue, 3))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.Poll(10*time.Millisecond, 200*time.Millisecond, func() (done bool, err error) {
		return e.claimedIDs() == 3, nil
	})
	a.NoError(err)

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
}

func TestSameJobNeverRunsConcurrently(t *testing.T) {
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              3,
		MaxJobRetries:        5,
		BaseRetryDelay:       10 * time.Millisecond,
		RetryJobAfterFailure: false,
		PerWorkerQPS:         10,
		Timeout:              time.Second,
	}

	e, err := New(c)
	a.NoError(err)

	// One trigger watching create, update and delete: three jobs sharing an ID,
	// each a read-modify-write like bump-application-revision.
	counter := &sharedCounter{}
	for i := 1; i <= 3; i++ {
		a.NoError(e.AddJob(&incrementingJob{counter}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.PollUntilContextTimeout(ctx, 5*time.Millisecond, 2*time.Second, true, func(_ context.Context) (bool, error) {
		return counter.get() == 3, nil
	})
	a.NoError(err, "every job should increment the counter once, got %d", counter.get())

	cancel()
	<-ch
}

func TestSameJobBurstRunsEveryJobInOrder(t *testing.T) {
	a := assert.New(t)
	c := Config{
		QueueSize:            10,
		Workers:              3,
		MaxJobRetries:        0,
		BaseRetryDelay:       10 * time.Millisecond,
		RetryJobAfterFailure: false,
		PerWorkerQPS:         100,
		Timeout:              time.Second,
	}

	e, err := New(c)
	a.NoError(err)

	// Waiting for a busy ID must not spend the retry budget, which is zero here.
	tr := &tracker{}
	var want []string
	for i := 1; i <= 6; i++ {
		name := fmt.Sprint(i)
		want = append(want, name)
		a.NoError(e.AddJob(&recordingJob{name: name, id: "same", duration: 20 * time.Millisecond, tracker: tr}))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.PollUntilContextTimeout(ctx, 5*time.Millisecond, 2*time.Second, true, func(_ context.Context) (bool, error) {
		return len(tr.ranJobs()) == len(want), nil
	})
	a.NoError(err)
	a.Equal(want, tr.ranJobs())
	a.Equal(1, tr.maxInflight())

	cancel()
	<-ch
}

func TestConcurrentJobDoesNotReleaseSharedID(t *testing.T) {
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              3,
		MaxJobRetries:        0,
		BaseRetryDelay:       10 * time.Millisecond,
		RetryJobAfterFailure: false,
		PerWorkerQPS:         100,
		Timeout:              time.Second,
	}

	e, err := New(c)
	a.NoError(err)

	tr := &tracker{}
	a.NoError(e.AddJob(&recordingJob{name: "slow", id: "same", duration: 200 * time.Millisecond, tracker: tr}))
	a.NoError(e.AddJob(&recordingJob{name: "concurrent", id: "same", concurrent: true, tracker: tr}))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.PollUntilContextTimeout(ctx, 5*time.Millisecond, time.Second, true, func(_ context.Context) (bool, error) {
		return slices.Contains(tr.ranJobs(), "concurrent"), nil
	})
	a.NoError(err)
	a.NoError(e.AddJob(&recordingJob{name: "late", id: "same", duration: 20 * time.Millisecond, tracker: tr}))

	err = wait.PollUntilContextTimeout(ctx, 5*time.Millisecond, time.Second, true, func(_ context.Context) (bool, error) {
		return len(tr.ranJobs()) == 3, nil
	})
	a.NoError(err)
	a.Equal([]string{"concurrent", "slow", "late"}, tr.ranJobs())
	a.Equal(1, tr.maxInflight())

	cancel()
	<-ch
}

func TestRetryKeepsItsPlaceInLine(t *testing.T) {
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              3,
		MaxJobRetries:        3,
		BaseRetryDelay:       20 * time.Millisecond,
		RetryJobAfterFailure: true,
		PerWorkerQPS:         100,
		Timeout:              time.Second,
	}

	e, err := New(c)
	a.NoError(err)

	tr := &tracker{}
	a.NoError(e.AddJob(&recordingJob{name: "flaky", id: "same", failures: 2, tracker: tr}))
	a.NoError(e.AddJob(&recordingJob{name: "next", id: "same", tracker: tr}))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.PollUntilContextTimeout(ctx, 5*time.Millisecond, 2*time.Second, true, func(_ context.Context) (bool, error) {
		return len(tr.ranJobs()) == 2, nil
	})
	a.NoError(err)
	a.Equal([]string{"flaky", "next"}, tr.ranJobs())
	a.Equal(1, tr.maxInflight())

	cancel()
	<-ch
}

func TestFailedRequeuing(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              1,
		MaxJobRetries:        3,
		BaseRetryDelay:       50 * time.Millisecond,
		RetryJobAfterFailure: true,
		PerWorkerQPS:         500,
		Timeout:              25 * time.Millisecond,
	}

	e, err := New(c)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 0))

	j1 := &failingJob{"1"}
	err = e.AddJob(j1)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 1))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	// Test the job itself failed
	// requeued 3 times (max retries)
	err = wait.Poll(1*time.Millisecond, 500*time.Millisecond, func() (done bool, err error) {
		return e.queue.NumRequeues(j1) == 3, nil
	})
	a.NoError(err, fmt.Sprint(e.queue.NumRequeues(j1)), "3")
	// too many requeues, cleared
	err = wait.Poll(1*time.Millisecond, 500*time.Millisecond, func() (done bool, err error) {
		return e.queue.NumRequeues(j1) == 0, nil
	})
	a.NoError(err, fmt.Sprint(e.queue.NumRequeues(j1)), "0")

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
}

func TestTimedOutRequeuing(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              1,
		MaxJobRetries:        3,
		BaseRetryDelay:       50 * time.Millisecond,
		RetryJobAfterFailure: true,
		PerWorkerQPS:         500,
		Timeout:              25 * time.Millisecond,
	}

	e, err := New(c)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 0))

	j2 := &sleepingJob{1 * time.Second, "2"}
	err = e.AddJob(j2)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 1))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	// Test the job timed out
	err = wait.Poll(10*time.Millisecond, 1000*time.Millisecond, func() (done bool, err error) {
		return e.queue.NumRequeues(j2) == 3, nil
	})
	a.NoError(err, fmt.Sprint(e.queue.NumRequeues(j2)), "3")
	// too many requeues, cleared
	err = wait.Poll(10*time.Millisecond, 1000*time.Millisecond, func() (done bool, err error) {
		return e.queue.NumRequeues(j2) == 0, nil
	})
	a.NoError(err, fmt.Sprint(e.queue.NumRequeues(j2)), "0")

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
}

type sleepingJob struct {
	duration time.Duration
	id       string
}

func (s *sleepingJob) ID() string {
	return s.duration.String() + s.id
}

func (s *sleepingJob) Run(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		time.Sleep(s.duration)
		close(ch)
	}()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return nil
	}
}

func (s *sleepingJob) AllowConcurrency() bool {
	return false
}

func (s *sleepingJob) Type() string {
	return "sleeping-job"
}

type failingJob struct {
	id string
}

func (f *failingJob) Type() string {
	return "failing-job"
}

func (f *failingJob) ID() string {
	return f.id
}

func (f *failingJob) Run(ctx context.Context) error {
	return fmt.Errorf("failing job %s is intended to fail ", f.id)
}

func (f *failingJob) AllowConcurrency() bool {
	return false
}

type sharedCounter struct {
	mu    sync.Mutex
	value int
}

func (c *sharedCounter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func (c *sharedCounter) set(v int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = v
}

// incrementingJob reads the counter and writes it back plus one, so overlapping
// runs lose increments.
type incrementingJob struct {
	counter *sharedCounter
}

func (i *incrementingJob) Type() string {
	return "incrementing-job"
}

func (i *incrementingJob) ID() string {
	return "incrementing"
}

func (i *incrementingJob) Run(_ context.Context) error {
	v := i.counter.get()
	time.Sleep(50 * time.Millisecond)
	i.counter.set(v + 1)
	return nil
}

func (i *incrementingJob) AllowConcurrency() bool {
	return false
}

// tracker records the order jobs finish in, and how many that disallow
// concurrency overlapped.
type tracker struct {
	mu       sync.Mutex
	inflight int
	max      int
	ran      []string
}

func (t *tracker) start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inflight++
	t.max = max(t.max, t.inflight)
}

func (t *tracker) end() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inflight--
}

func (t *tracker) record(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ran = append(t.ran, name)
}

func (t *tracker) ranJobs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.ran)
}

func (t *tracker) maxInflight() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.max
}

type recordingJob struct {
	name       string
	id         string
	duration   time.Duration
	concurrent bool
	failures   int
	tracker    *tracker
}

func (r *recordingJob) Type() string {
	return "recording-job"
}

func (r *recordingJob) ID() string {
	return r.id
}

func (r *recordingJob) Run(_ context.Context) error {
	if !r.concurrent {
		r.tracker.start()
		defer r.tracker.end()
	}
	time.Sleep(r.duration)
	if r.failures > 0 {
		r.failures--
		return fmt.Errorf("recording job %s is intended to fail", r.name)
	}
	r.tracker.record(r.name)
	return nil
}

func (r *recordingJob) AllowConcurrency() bool {
	return r.concurrent
}

// claimedIDs counts the IDs held by jobs that disallow concurrency.
func (e *Executor) claimedIDs() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.lines)
}

func waitForAdded(q workqueue.TypedDelayingInterface[Job], depth int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := wait.PollUntilContextTimeout(ctx, 1*time.Millisecond, 1*time.Second, true, func(ctx context.Context) (done bool, err error) {
		if q.Len() == depth {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		fmt.Printf("%d != %d", q.Len(), depth)
	}
	return err
}
