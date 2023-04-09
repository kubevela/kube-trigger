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

package cronjob

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/types"
)

var (
	cronJobType = "cronjob"
)

// CronJob triggers Actions on a schedule.
type CronJob struct {
	config     Config
	cronRunner *cron.Cron
}

var _ types.Source = &CronJob{}

// New creates a new CronJob.
func (c *CronJob) New() types.Source {
	return &CronJob{}
}

// Init initializes the CronJob.
func (c *CronJob) Init(properties *runtime.RawExtension, eh eventhandler.EventHandler) error {
	b, err := properties.MarshalJSON()
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties for %s", c.Type())
	}
	err = json.Unmarshal(b, &c.config)
	if err != nil {
		return errors.Wrapf(err, "error when parsing properties for %s", c.Type())
	}

	c.cronRunner = cron.New()
	sched, err := cron.ParseStandard(formatSchedule(c.config))
	if err != nil {
		return errors.Wrapf(err, "error when parsing schedule for %s", c.Type())
	}
	c.cronRunner.Schedule(sched, cron.FuncJob(func() {
		logger.Infof("schedule \"%s\" fired", c.config.String())
		e := Event{
			Config:    c.config,
			TimeFired: metav1.Now(),
		}
		err := eh(c.Type(), e, e)
		if err != nil {
			logger.Infof("calling event handler failed: %s", err)
		}
	}))

	return nil
}

// Run starts the CronJob.
func (c *CronJob) Run(ctx context.Context) error {
	go func() {
		logger.Infof("cronjob \"%s\" started", c.config.String())
		c.cronRunner.Start()
		<-ctx.Done()
		logger.Infof("context cancelled, stoppping cronjob \"%s\"", c.config.String())
		c.cronRunner.Stop()
	}()

	return nil
}

// Type returns the type of the CronJob.
func (c *CronJob) Type() string {
	return cronJobType
}

// Singleton .
func (c *CronJob) Singleton() bool {
	return false
}
