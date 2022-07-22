package handler

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevela/kube-trigger/pkg/api"
	"github.com/kubevela/kube-trigger/pkg/event"
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
	Filters []api.Filter `json:"filters" yaml:"filters"`
	To      api.App      `json:"to" yaml:"to"`
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
	logrus.Infof("[%s] got message: %s", id, e.Message())
	ctx := context.WithValue(context.Background(), "id", id)

	var err error
	defer func() {
		if err != nil {
			logrus.Infof("[%s] send event err: %v", id, err)
		}
	}()

	for _, f := range d.Filters {
		if f.Name != "" && e.Obj.GetName() != f.Name {
			return
		}
	}

	if d.To.Name != "" {
		var app v1beta1.Application
		err = d.Client.Get(ctx, client.ObjectKey{Name: d.To.Name, Namespace: d.To.Namespace}, &app)
		if err != nil {
			return
		}
		app.Annotations["app.oam.dev/publishVersion"] = id
		err = d.Client.Update(ctx, &app)
		return
	}

}
