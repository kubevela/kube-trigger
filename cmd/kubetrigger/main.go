package main

import (
	"flag"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/kubevela/kube-trigger/pkg/api"
	"github.com/kubevela/kube-trigger/pkg/controller"
	"github.com/kubevela/kube-trigger/pkg/filter/registry"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	registry.RegisterBuiltinFilters()

	triggerPath := flag.String("config", "examples/trigger.yaml", "specify the config path of the trigger")
	flag.Parse()

	data, err := ioutil.ReadFile(*triggerPath)
	if err != nil {
		logrus.Fatal("read file", *triggerPath, err)
	}
	tr := api.Trigger{}
	err = yaml.Unmarshal(data, &tr)
	if err != nil {
		logrus.Fatal("unmarshal config file", *triggerPath, err)
	}

	controller.Start(&tr)
}
