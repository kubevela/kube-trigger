package cronjob

import (
	"context"
	"encoding/json"

	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	cronJobType = "cronjob"
)

type CronJob struct {
	config     Config
	cronRunner *cron.Cron
}

var _ types.Source = &CronJob{}

func (c *CronJob) New() types.Source {
	return &CronJob{}
}

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
			logger.Warnf("calling event handler failed: %s", err)
		}
	}))

	return nil
}

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

func (c *CronJob) Type() string {
	return cronJobType
}

func (c *CronJob) Singleton() bool {
	return false
}
