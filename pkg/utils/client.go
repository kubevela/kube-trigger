package utils

import (
	"github.com/oam-dev/kubevela/apis/core.oam.dev/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClient *client.Client

func GetClient() (*client.Client, error) {
	if k8sClient != nil {
		return k8sClient, nil
	}

	conf := ctrl.GetConfigOrDie()
	err := v1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	c, err := client.New(conf, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	k8sClient = &c
	return k8sClient, nil
}
