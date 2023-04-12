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
	// maxRetryDelay is the max re-queueing delay of the exponential failure
	// rate limiter.
	maxRetryDelay = 1200 * time.Second
)

// Executor is a rate-limited job queue with concurrent workers.
type Executor struct {
	workers      int
	maxQueueSize int
	maxRetries   int
	allowRetries bool
	wg           sync.WaitGroup
	timeout      time.Duration
	logger       *logrus.Entry
	runningJobs  sync.Map
	queue        workqueue.RateLimitingInterface
}

// Job is an Action to be executed by the workers in the Executor.
type Job interface {
	Type() string
	ID() string
	Run(ctx context.Context) error
	AllowConcurrency() bool
}

// Config is the config for the Executor.
type Config struct {
	// QueueSize is the maximum number of jobs in the executor queue.
	QueueSize int
	// Workers is the number of workers that are concurrently executing jobs.
	Workers int
	// MaxJobRetries is the maximum number of retries if a job fails. This should
	// not be zero. If you want to disable retries, just disable RetryJobAfterFailure.
	MaxJobRetries int
	// BaseRetryDelay defines how long after a job fails before it is re-queued.
	BaseRetryDelay time.Duration
	// RetryJobAfterFailure allows the job to be re-queued if it fails.
	RetryJobAfterFailure bool
	// PerWorkerQPS is the max QPS of a worker before it is rate-limited. With Workers,
	// Workers*PerWorkerQPS is the overall QPS limit of the entire executor.
	PerWorkerQPS int
	// Timeout defines how long a single job is allowed to run and how long the
	// entire executor should wait for all the jobs to stop when shutting down.
	Timeout time.Duration
}

// New creates a new Executor with user-provided Config.
func New(c Config) (*Executor, error) {
	if c.QueueSize == 0 || c.Workers == 0 || c.BaseRetryDelay == 0 ||
		c.Timeout == 0 || c.PerWorkerQPS == 0 {
		return nil, fmt.Errorf("invalid executor config")
	}
	e := &Executor{}
	e.workers = c.Workers
	e.timeout = c.Timeout
	e.maxQueueSize = c.QueueSize
	e.maxRetries = c.MaxJobRetries
	e.allowRetries = c.RetryJobAfterFailure
	e.wg = sync.WaitGroup{}
	e.runningJobs = sync.Map{}
	// Create a rate limited queue, with a token bucket for overall limiting,
	// and exponential failure for per-item limiting.
	e.queue = workqueue.NewRateLimitingQueue(
		workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(c.BaseRetryDelay, maxRetryDelay),
			&workqueue.BucketRateLimiter{
				// Token Bucket limiter, with
				// qps = workers * qpsToWorkerRatio, maxBurst = QueueSize
				Limiter: rate.NewLimiter(rate.Limit(c.Workers*c.PerWorkerQPS), c.QueueSize),
			},
		),
	)
	e.logger = logrus.WithField("executor", "action-job-executor")
	e.logger.Infof("new executor created, %d queue size, %d concurrnt workers, %v timeout, "+
		"allow retries %v, max %d retries",
		e.maxQueueSize,
		e.workers,
		e.timeout,
		e.allowRetries,
		e.maxRetries,
	)
	return e, nil
}

func (e *Executor) setJobStatus(j Job, status bool) {
	if status {
		e.runningJobs.Store(j.ID(), true)
	} else {
		e.runningJobs.Delete(j.ID())
	}
}

func (e *Executor) setJobRunning(j Job) {
	e.setJobStatus(j, true)
}

func (e *Executor) setJobFinished(j Job) {
	e.setJobStatus(j, false)
}

func (e *Executor) getJobStatus(j Job) bool {
	v, ok := e.runningJobs.Load(j.ID())
	if !ok {
		return false
	}
	return v.(bool)
}

func (e *Executor) requeueJob(j Job) {
	if e.queue.NumRequeues(j) < e.maxRetries {
		e.queue.AddRateLimited(j)
		return
	}
	e.logger.Errorf("job %s (%s) cannot be requeued because it failed too many (%d/%d) times", j.Type(), j.ID(), e.queue.NumRequeues(j), e.maxRetries)
	e.queue.Forget(j)
}

// AddJob adds a job to the queue.
func (e *Executor) AddJob(j Job) error {
	if e.queue.Len() >= e.maxQueueSize {
		msg := fmt.Sprintf("job %s (%s) cannot be added, queue size full %d/%d", j.Type(), j.ID(), e.queue.Len(), e.maxQueueSize)
		e.logger.Errorf(msg)
		return fmt.Errorf(msg)
	}
	e.queue.Add(j)
	e.logger.Debugf("job %s (%s) added to queue, currnet queue size: %d/%d", j.Type(), j.ID(), e.queue.Len(), e.maxQueueSize)
	return nil
}

func (e *Executor) runJob(ctx context.Context) bool {
	if ctx.Err() != nil {
		e.logger.Infof("worker exiting because %s", ctx.Err())
		return false
	}

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

	e.logger.Debugf("job %s (%s) is picked up by a worker", j.Type(), j.ID())

	// This job does not allow concurrent runs, and it is already running.
	// Requeue it to run it later.
	if !j.AllowConcurrency() && e.getJobStatus(j) {
		e.logger.Infof("job %s (%s) is already running, will be requeued", j.Type(), j.ID())
		e.requeueJob(j)
		return true
	}

	// Add a job timeout
	timeoutCtx, cancel := context.WithDeadline(ctx, time.Now().Add(e.timeout))
	defer cancel()

	e.logger.Infof("job %s (%s) started executing", j.Type(), j.ID())
	e.setJobRunning(j)
	err := j.Run(timeoutCtx)
	e.setJobFinished(j)

	if err == nil && timeoutCtx.Err() == nil {
		e.logger.Infof("job %s (%s) finished", j.Type(), j.ID())
		e.queue.Forget(j)
		return true
	}

	// context cancelled, it is time to die
	if timeoutCtx.Err() == context.Canceled {
		e.logger.Infof("job %s (%s) failed because ctx errored: %s, worker will exit soon", j.Type(), j.ID(), timeoutCtx.Err())
		return false
	}

	msg := fmt.Sprintf("job %s (%s) failed because (jobErr=%v, ctxErr=%v)", j.Type(), j.ID(), err, timeoutCtx.Err())
	if e.allowRetries {
		msg += fmt.Sprintf(", will retry job %s (%s) later", j.Type(), j.ID())
		e.requeueJob(j)
	}
	e.logger.Errorf(msg)

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
	e.logger.Infof("executor started with %d workers", e.workers)

	<-ctx.Done()

	e.logger.Infof("shutting down executor")

	// Shutdown queue, wait for workers to end with a timeout.
	e.wg.Add(1)
	go func() {
		e.queue.ShutDownWithDrain()
		e.wg.Done()
	}()
	ch := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		e.logger.Infof("shutdown successful")
	case <-time.After(e.timeout):
		e.logger.Infof("shutdown timed out")
	}
}
