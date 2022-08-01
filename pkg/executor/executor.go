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
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
)

const (
	maxRetryDelay    = 1200 * time.Second
	qpsToWorkerRatio = 2
)

// Executor is a rate-limited work queue with concurrent workers.
type Executor struct {
	workers      int
	queueSize    int
	maxRetries   int
	allowRetries bool
	lock         sync.RWMutex
	wg           sync.WaitGroup
	timeout      time.Duration
	logger       *logrus.Entry
	runningJobs  map[string]bool
	queue        workqueue.RateLimitingInterface
}

// Job is an Action to be executed by the workers in the Executor.
type Job interface {
	Type() string
	Run(ctx context.Context) error
	AllowConcurrency() bool
}

type Config struct {
	QueueSize            int
	Workers              int
	MaxJobRetries        int
	BaseRetryDelay       time.Duration
	RetryJobAfterFailure bool
	Timeout              time.Duration
}

// New creates a new Executor with a queue size, number of workers,
// and a job-running or shutdown timeout.
func New(c Config) (*Executor, error) {
	if c.QueueSize == 0 || c.Workers == 0 || c.MaxJobRetries == 0 ||
		c.BaseRetryDelay == 0 || c.Timeout == 0 {
		return nil, fmt.Errorf("invalid executor config")
	}
	e := &Executor{}
	e.workers = c.Workers
	e.timeout = c.Timeout
	e.queueSize = c.QueueSize
	e.maxRetries = c.MaxJobRetries
	e.allowRetries = c.RetryJobAfterFailure
	e.wg = sync.WaitGroup{}
	e.runningJobs = make(map[string]bool)
	e.lock = sync.RWMutex{}
	// Create a rate limited queue, with a token bucket for overall limiting,
	// and exponential failure for per-item limiting.
	e.queue = workqueue.NewRateLimitingQueue(
		workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(c.BaseRetryDelay, maxRetryDelay),
			&workqueue.BucketRateLimiter{
				// Token Bucket limiter, with
				// qps = workers * qpsToWorkerRatio, maxBurst = queueSize
				Limiter: rate.NewLimiter(rate.Limit(c.Workers*qpsToWorkerRatio), c.QueueSize),
			},
		),
	)
	e.logger = logrus.WithField("executor", "action-job-executor")
	e.logger.Infof("new executor created, %d queue size, %d concurrnt workers, %v timeout, "+
		"allow retries %v, max %d retries",
		e.queueSize,
		e.workers,
		e.timeout,
		e.allowRetries,
		e.maxRetries,
	)
	return e, nil
}

func (e *Executor) setJobStatus(j Job, status bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.runningJobs[j.Type()] = status
}

func (e *Executor) setJobRunning(j Job) {
	e.setJobStatus(j, true)
}

func (e *Executor) setJobNotRunning(j Job) {
	e.setJobStatus(j, false)
}

func (e *Executor) getJobStatus(j Job) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.runningJobs[j.Type()]
}

func (e *Executor) requeueJob(j Job) {
	if e.queue.NumRequeues(j) < e.maxRetries {
		e.queue.AddRateLimited(j)
		return
	}
	e.logger.Errorf("requeue job %s failed, it failed %d times, too many retries", j.Type(), e.queue.NumRequeues(j))
	e.queue.Forget(j)
}

// AddJob adds a job to the queue.
func (e *Executor) AddJob(j Job) error {
	if e.queue.Len() >= e.queueSize {
		return fmt.Errorf("queue full with size %d, cannot add job %s", e.queue.Len(), j.Type())
	}
	e.queue.Add(j)
	return nil
}

func (e *Executor) runJob(ctx context.Context) bool {
	item, quit := e.queue.Get()
	if quit {
		return false
	}

	defer e.queue.Done(item)

	j, ok := item.(Job)
	if !ok {
		return true
	}
	if j == nil {
		return true
	}

	e.logger.Infof("job picked up by a worker, going running job: %s", j.Type())

	// This job does not allow concurrent runs, and it is already running.
	// Requeue it to run it later.
	if !j.AllowConcurrency() && e.getJobStatus(j) {
		e.logger.Infof("same job %s is already running, will run later", j.Type())
		e.requeueJob(j)
		return true
	}

	// Add a job timeout
	timeoutCtx, cancel := context.WithDeadline(ctx, time.Now().Add(e.timeout))
	defer cancel()

	e.setJobRunning(j)
	err := j.Run(timeoutCtx)
	e.setJobNotRunning(j)

	if err == nil && timeoutCtx.Err() == nil {
		e.logger.Infof("job %s finished", j.Type())
		e.queue.Forget(j)
	} else {
		e.logger.Errorf("job %s failed: jobErr=%s, ctxErr=%s", j.Type(), err, timeoutCtx.Err())
		if e.allowRetries {
			e.logger.Infof("will retry job %s later", j.Type())
			e.requeueJob(j)
		}
	}

	return true
}

// RunJobs starts workers.
func (e *Executor) RunJobs(ctx context.Context) {
	e.wg.Add(e.workers)
	for i := 0; i < e.workers; i++ {
		go func() {
			for e.runJob(ctx) {
			}
			e.wg.Done()
		}()
	}
}

// Shutdown stops workers.
func (e *Executor) Shutdown() bool {
	e.logger.Infof("shutting down executor")

	// Wait for workers to end with a timeout.
	ch := make(chan struct{})
	go func() {
		e.queue.ShutDown()
		e.wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		e.logger.Infof("shutdown successful")
		return true
	case <-time.After(e.timeout):
		e.logger.Infof("shutdown timed out")
		return false
	}
}
