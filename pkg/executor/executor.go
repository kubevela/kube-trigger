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
)

// Executor is a job queue with multiple workers that can run concurrently.
type Executor struct {
	jobChan        chan Job
	wg             sync.WaitGroup
	lock           sync.RWMutex
	maxConcurrency int
	timeout        time.Duration
	runningJobs    map[string]bool
	logger         *logrus.Entry
}

// Job is an Action to be executed by the workers in the Executor.
type Job interface {
	Type() string
	Run(ctx context.Context) error
	AllowConcurrency() bool
}

// New creates a new Executor with a queue size, number of workers,
// and a job-running or shutdown timeout.
func New(queueSize int, maxConcurrency int, timeout time.Duration) *Executor {
	e := &Executor{}
	e.lock = sync.RWMutex{}
	e.runningJobs = make(map[string]bool)
	e.maxConcurrency = maxConcurrency
	e.timeout = timeout
	e.jobChan = make(chan Job, queueSize)
	e.logger = logrus.WithField("executor", "action-job-executor")
	e.logger.Infof("new executor created, %d queue size, %d concurrnt workers, %v timeout",
		queueSize,
		maxConcurrency,
		timeout)
	return e
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

// AddJob adds a job to the queue.
func (e *Executor) AddJob(j Job) error {
	select {
	case e.jobChan <- j:
		e.logger.Infof("added job to executor: %s", j.Type())
		return nil
	default:
		return fmt.Errorf("job queue full, cannot add job %s", j.Type())
	}
}

//nolint:gocognit
func (e *Executor) runJob(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-e.jobChan:
			// Make sure it exits when context is canceled, even if there are jobs left.
			if ctx.Err() != nil {
				return
			}
			if j == nil {
				return
			}
			e.logger.Infof("job picked up by a worker, running job: %s", j.Type())
			// This job does not allow concurrent runs, and it is already running.
			// Requeue it to run it later.
			if !j.AllowConcurrency() && e.getJobStatus(j) {
				err := e.AddJob(j)
				if err != nil {
					e.logger.Errorf("requeueing job %s failed: %s", j.Type(), err)
				}
				return
			}
			e.setJobRunning(j)
			// Add a job timeout
			timeoutCtx, cancel := context.WithDeadline(ctx, time.Now().Add(e.timeout))
			err := j.Run(timeoutCtx)
			e.setJobNotRunning(j)
			if err == nil && timeoutCtx.Err() == nil {
				e.logger.Infof("job %s finished", j.Type())
			} else {
				e.logger.Errorf("job %s failed: jobErr=%s, ctxErr=%s", j.Type(), err, timeoutCtx.Err())
			}
			// Avoid context leak.
			cancel()
		}
	}
}

// RunJobs starts workers.
func (e *Executor) RunJobs(ctx context.Context) {
	e.wg.Add(e.maxConcurrency)
	for i := 0; i < e.maxConcurrency; i++ {
		go e.runJob(ctx)
	}
}

// Shutdown stops workers.
func (e *Executor) Shutdown() bool {
	close(e.jobChan)
	e.logger.Infof("shutting down executor")

	// Wait for workers to end with a timeout.
	ch := make(chan struct{})
	go func() {
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
