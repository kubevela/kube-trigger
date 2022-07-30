package handler

// TODO: delete this file, should be fine, code in unused

import (
	"math/rand"
	"strconv"
	"time"

	actiontypes "github.com/kubevela/kube-trigger/pkg/action/types"
	filtertypes "github.com/kubevela/kube-trigger/pkg/filter/types"
	"github.com/kubevela/kube-trigger/pkg/source/builtin/k8sresourcewatcher/event"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Trigger is implemented by any handler.
// The Handle method is used to process event
type Trigger interface {
	Init() error
	Handle(e event.Event)
}

// Map maps each event handler function to a name for easily lookup
var Map = map[string]interface{}{
	"application": &AppTrigger{},
}

// AppTrigger handler implements Trigger interface,
// print each event with JSON format
type AppTrigger struct {
	Filters []filtertypes.FilterMeta
	Actions []actiontypes.ActionMeta
	Client  client.Client
}

// Init initializes handler configuration
// Do nothing for default handler
func (d *AppTrigger) Init() error {
	return nil
}

// Handle handles an event.
func (d *AppTrigger) Handle(e event.Event) {
	rand.Seed(time.Now().UnixNano())
	id := strconv.FormatInt(rand.Int63(), 10)
	// logrus.Infof("[%s] got message: %s", id, e.Message())
	//ctx := context.WithValue(context.Background(), "id", id)

	var err error
	defer func() {
		if err != nil {
			logrus.Infof("[%s] send event err: %v", id, err)
		}
	}()

	//// Apply filters
	//for _, f := range d.Filters {
	//	// TODO(charlie0129): use a cache
	//	// Currently it instantiates a Filter everytime.
	//	// This will have performance issues. I have already defined a
	//	// CachedRegistry to store cached filters. Use the CachedRegistry
	//	// to store instantiated filter for later use.
	//	fr, ok := registry.TypeRegistry.Find(f.Type)
	//	if !ok {
	//		// TODO(charlie0129): make this return an error instead of log here
	//		// The user have used a undefined filter. This should return an error.
	//		logrus.Errorf("filter type %s not found", f.Type)
	//		return
	//	}
	//	// TODO(charlie0129): use better variable names
	//	fi := fr.New()
	//	err := fi.Init(f.Properties)
	//	if err != nil {
	//		logrus.Errorf("%v", err)
	//		return
	//	}
	//	keep, err := fi.ApplyToObject(e.Obj)
	//	if err != nil {
	//		logrus.Errorf("%v", err)
	//	}
	//	if !keep {
	//		logrus.Debugf("filtered out: %s", e.Message())
	//		return
	//	}
	//	logrus.Debugf("kept: %s", e.Message())
	//}
	//
	//if d.To.Name != "" {
	//	var app v1beta1.Application
	//	err = d.Client.Get(ctx, client.ObjectKey{Name: d.To.Name, Namespace: d.To.Namespace}, &app)
	//	if err != nil {
	//		return
	//	}
	//	app.Annotations["app.oam.dev/publishVersion"] = id
	//	err = d.Client.Update(ctx, &app)
	//	return
	//}

}
