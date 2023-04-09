package cronjob

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Event struct {
	Config    `json:",inline"`
	TimeFired metav1.Time `json:"timeFired"`
}
