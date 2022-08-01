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

	actionregistry "github.com/kubevela/kube-trigger/pkg/action/registry"
	"github.com/kubevela/kube-trigger/pkg/config"
	"github.com/kubevela/kube-trigger/pkg/eventhandler"
	"github.com/kubevela/kube-trigger/pkg/executor"
	filterregistry "github.com/kubevela/kube-trigger/pkg/filter/registry"
	sourceregistry "github.com/kubevela/kube-trigger/pkg/source/registry"
	"github.com/kubevela/kube-trigger/pkg/source/types"
	"github.com/sirupsen/logrus"
)

func main() {
	// TODO(charlie0129): use a proper way to start. Currently, it is a disaster, full of testing code.

	logrus.SetLevel(logrus.InfoLevel)

	triggerPath := flag.String("config", "examples/sampleconf.cue", "specify the config path of the trigger")
	flag.Parse()

	data, err := ioutil.ReadFile(*triggerPath)
	if err != nil {
		logrus.Fatal("read file", *triggerPath, err)
	}

	//nolint
	exe := executor.New(20, 2, time.Second*5)

	sourceReg := sourceregistry.NewWithBuiltinSources()
	//nolint:gomnd
	filterReg := filterregistry.NewWithBuiltinFilters(10)
	//nolint:gomnd
	actionReg := actionregistry.NewWithBuiltinActions(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := &config.Config{}
	err = conf.Parse(string(data))
	if err != nil {
		//nolint
		os.Exit(1)
		return
	}

	var sources []types.Source

	// Initialize all watchers (sources)
	for _, w := range conf.Watchers {
		// Check if this source type exists
		s, exist := sourceReg.Get(w.Source)
		if !exist {
			//nolint
			os.Exit(1)
			return
		}

		newSource := s.New()

		// New EventHandler, this is called when an event happens.
		eh := eventhandler.New().
			// Add filters to it.
			WithFilters(w.Filters, filterReg).
			// Add action jobs to executor.
			WithActions(exe, w.Actions, actionReg)

		// Initialize source and add EventHandler to it
		err = newSource.Init(w.Source.Properties, eh)
		if err != nil {
			//nolint
			fmt.Println(err)
			return
		}

		sources = append(sources, newSource)
	}

	// Run sources
	for _, s := range sources {
		s := s
		go func() {
			//nolint
			err := s.Run(ctx)
			if err != nil {
				//nolint
				fmt.Println(err)
			}
		}()
	}

	exe.RunJobs(ctx)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm

	exe.Shutdown()
}
