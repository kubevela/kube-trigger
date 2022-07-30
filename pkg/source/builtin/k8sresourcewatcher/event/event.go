package event

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Event represent an event got from k8s api server
type Event struct {
	Namespace  string
	Kind       string
	ApiVersion string
	Name       string
	Info       string
	Obj        metav1.Object
}

// Message returns event message in standard format.
func (e *Event) Message() (msg string) {
	return "apiVersion: " + e.ApiVersion + ", kind: " + e.Kind + ", namespace: " + e.Namespace + ", name: " + e.Name + ", info: " + e.Info
}

// InformerEvent indicate the informerEvent
type InformerEvent struct {
	Key          string
	EventType    string
	Namespace    string
	ResourceType string
}
