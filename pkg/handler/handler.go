package handler


import (
"github.com/bitnami-labs/kubewatch/config"
"github.com/kubevela/kube-trigger/pkg/event"
"github.com/bitnami-labs/kubewatch/pkg/handlers/flock"
"github.com/bitnami-labs/kubewatch/pkg/handlers/hipchat"
"github.com/bitnami-labs/kubewatch/pkg/handlers/mattermost"
"github.com/bitnami-labs/kubewatch/pkg/handlers/msteam"
"github.com/bitnami-labs/kubewatch/pkg/handlers/slack"
"github.com/bitnami-labs/kubewatch/pkg/handlers/smtp"
"github.com/bitnami-labs/kubewatch/pkg/handlers/webhook"
)

// Handler is implemented by any handler.
// The Handle method is used to process event
type Handler interface {
	Init(c *config.Config) error
	Handle(e event.Event)
}

// Map maps each event handler function to a name for easily lookup
var Map = map[string]interface{}{
	"default":    &Default{},

}

// Default handler implements Handler interface,
// print each event with JSON format
type Default struct {
}

// Init initializes handler configuration
// Do nothing for default handler
func (d *Default) Init(c *config.Config) error {
	return nil
}

// Handle handles an event.
func (d *Default) Handle(e event.Event) {
}
