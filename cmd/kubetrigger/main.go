package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kubevela/kube-trigger/pkg/action/executor"
	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/config"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/sirupsen/logrus"
)

func main() {
	// TODO(charlie0129): use a proper way to start. Currently, it is a disaster, full of testing code.

	logrus.SetLevel(logrus.DebugLevel)

	triggerPath := flag.String("config", "examples/sampleconf.cue", "specify the config path of the trigger")
	flag.Parse()

	data, err := ioutil.ReadFile(*triggerPath)
	if err != nil {
		logrus.Fatal("read file", *triggerPath, err)
	}

	conf := &config.Config{}
	err = conf.Parse(string(data))
	if err != nil {
		return
	}

	exe := executor.New(20, 2, time.Second*5)

	sourceReg := sourceregistry.NewWithBuiltinSources()
	filterReg := filterregistry.NewWithBuiltinFilters()
	actionReg := actionregistry.NewWithBuiltinActions()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var sources []types.Source

	for _, w := range conf.Watchers {
		s, exist := sourceReg.GetType(w.Source)
		if !exist {
			os.Exit(1)
			return
		}

		newSource := s.New()

		err := newSource.Init(w.Source.Properties, w.Filters, filterReg)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, a := range w.Actions {
			newSource.AddEventHandler(executor.NewSourceEventHandler(exe, actionReg, a))
		}

		sources = append(sources, newSource)
	}

	for _, s := range sources {
		go s.Run(ctx)
	}

	exe.RunJobs(ctx)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}
