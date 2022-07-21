package event

import (
	"fmt"

	"github.com/bitnami-labs/kubewatch/pkg/utils"
	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	api_v1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1beta1 "k8s.io/api/rbac/v1beta1"
)

// Event represent an event got from k8s api server
type Event struct {
	Namespace string
	Kind      string
	Component string
	Host      string
	Reason    string
	Status    string
	Name      string
}

var m = map[string]string{
	"created": "Normal",
	"deleted": "Danger",
	"updated": "Warning",
}

// New create new KubewatchEvent
func New(obj interface{}, action string) Event {
	var namespace, kind, component, host, reason, status, name string

	objectMeta := utils.GetObjectMetaData(obj)
	namespace = objectMeta.Namespace
	name = objectMeta.Name
	reason = action
	status = m[action]

	switch object := obj.(type) {

	case Event:
		name = object.Name
		kind = object.Kind
		namespace = object.Namespace
	}

	kbEvent := Event{
		Namespace: namespace,
		Kind:      kind,
		Component: component,
		Host:      host,
		Reason:    reason,
		Status:    status,
		Name:      name,
	}
	return kbEvent
}

// Message returns event message in standard format.
func (e *Event) Message() (msg string) {

	return ""
}
