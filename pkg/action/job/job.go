package job

import (
	"context"

	"github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/action/types"
)

type Job struct {
	action     types.Action
	sourceType string
	event      interface{}
}

// New creates a new job. It will fetch cached Action instance from Registry
// using provided ActionMeta. sourceType and event will be passed to the Action.Run
// method.
func New(reg *registry.Registry, meta types.ActionMeta, sourceType string, event interface{}) (*Job, error) {
	var err error
	ret := Job{
		sourceType: sourceType,
		event:      event,
	}
	ret.action, err = reg.CreateOrGetInstance(meta)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (j *Job) Type() string {
	if j.action == nil {
		return ""
	}
	return j.action.Type()
}

func (j *Job) Run(ctx context.Context) error {
	if j.action == nil {
		return nil
	}
	return j.action.Run(ctx, j.sourceType, j.event)
}

func (j *Job) AllowConcurrency() bool {
	if j.action == nil {
		return true
	}
	return j.action.AllowConcurrency()
}
