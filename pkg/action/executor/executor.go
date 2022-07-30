package executor

import (
	"context"
	"sync"
	"time"

	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
	sourcetypes "github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/sirupsen/logrus"
)

type Job struct {
	action     types.Action
	sourceType string
	event      interface{}
}

func NewSourceEventHandler(executor *Executor, reg *actionregistry.Registry, meta types.ActionMeta) sourcetypes.EventHandler {
	return func(sourceType string, event interface{}) {
		j := NewJob(reg, meta, sourceType, event)
		if j.action == nil {
			return
		}
		ok := executor.AddJob(j)
		if !ok {
			logrus.Error("job queue full") // TODO: more info
		}
	}
}

func NewJob(reg *actionregistry.Registry, meta types.ActionMeta, sourceType string, event interface{}) Job {
	var err error
	ret := Job{
		sourceType: sourceType,
		event:      event,
	}
	ret.action, err = reg.CreateOrGetInstance(meta)
	if err != nil {
		logrus.Errorf("action type %s not found or cannot create instance: %s", meta.Type, err)
		return Job{}
	}
	return ret
}

func (j *Job) Run(ctx context.Context) error {
	if j.action == nil {
		return nil
	}
	return j.action.Run(ctx, j.sourceType, j.event)
}

type Executor struct {
	jobChan         chan Job
	wg              sync.WaitGroup
	maxConcurrency  int
	shutdownTimeout time.Duration
	logger          *logrus.Entry
}

func New(queueSize int, maxConcurrency int, shutdownTimeout time.Duration) *Executor {
	e := &Executor{}
	e.maxConcurrency = maxConcurrency
	e.shutdownTimeout = shutdownTimeout
	e.jobChan = make(chan Job, queueSize)
	e.logger = logrus.WithField("executor", "action")
	e.logger.Debugf("new executor created with queueSize=%d, maxConcurrency=%d, shutdownTimeout=%v", queueSize, maxConcurrency, shutdownTimeout)
	return e
}

func (e *Executor) AddJob(j Job) bool {
	select {
	case e.jobChan <- j:
		e.logger.Debugf("added job to executor: actionType=%s", j.action.Type())
		return true
	default:
		e.logger.Errorf("job queue full, add failed: actionType=%s", j.action.Type())
		return false
	}
}

func (e *Executor) runJob(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-e.jobChan:
			e.logger.Debugf("running job: actionType=%s", j.action.Type())
			err := j.Run(ctx)
			if err != nil {
				e.logger.Errorf("job: actionType=%s failed: %s", j.action.Type(), err.Error())
			}
			// Make sure it exits, after at most one last job
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func (e *Executor) RunJobs(ctx context.Context) {
	e.wg.Add(e.maxConcurrency)
	for i := 0; i < e.maxConcurrency; i++ {
		go e.runJob(ctx)
	}
}

func (e *Executor) Shutdown() bool {
	close(e.jobChan)

	ch := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		return true
	case <-time.After(e.shutdownTimeout):
		return false
	}
}
