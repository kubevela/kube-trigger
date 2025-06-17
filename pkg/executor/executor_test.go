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
		l := 0
		e.runningJobs.Range(func(_ any, _ any) bool {
			l += 1
			return true
		})
		return l == 3, nil
	})
	a.NoError(err)

	// Wait for all jobs to end
	err = wait.Poll(1*time.Millisecond, 200*time.Millisecond, func() (done bool, err error) {
		l := 0
		e.runningJobs.Range(func(_ any, _ any) bool {
			l += 1
			return true
		})
		return l == 0, nil
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
		l := 0
		e.runningJobs.Range(func(_ any, _ any) bool {
			l += 1
			return true
		})
		return l == 3, nil
	})
	a.NoError(err)

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
}

func TestSameJobRequeuing(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	a := assert.New(t)
	c := Config{
		QueueSize:            5,
		Workers:              3,
		MaxJobRetries:        5,
		BaseRetryDelay:       10 * time.Millisecond,
		RetryJobAfterFailure: true,
		PerWorkerQPS:         10,
		Timeout:              300 * time.Millisecond,
	}

	e, err := New(c)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 0))

	// Jobs with same id
	j1 := &sleepingJob{200 * time.Millisecond, "1"}
	j2 := &sleepingJob{200 * time.Millisecond, "1"}
	j3 := &sleepingJob{100 * time.Millisecond, "1"}

	err = e.AddJob(j1)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 1))

	err = e.AddJob(j2)
	a.NoError(err)
	a.NoError(waitForAdded(e.queue, 2))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan struct{})
	go func() {
		e.RunJobs(ctx)
		close(ch)
	}()

	err = wait.Poll(1*time.Millisecond, 200*time.Millisecond, func() (done bool, err error) {
		l := 0
		e.runningJobs.Range(func(_ any, _ any) bool {
			l += 1
			return true
		})
		return l >= 1, nil
	})
	a.NoError(err)

	err = e.AddJob(j3)
	a.NoError(err)

	err = wait.Poll(1*time.Millisecond, 500*time.Millisecond, func() (done bool, err error) {
		return e.queue.NumRequeues(j3) >= 2, nil
	})

	cancel()
	<-ch
	a.NoError(waitForAdded(e.queue, 0))
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
