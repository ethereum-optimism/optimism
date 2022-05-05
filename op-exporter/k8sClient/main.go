package k8sClient

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Newk8sClient() (client *kubernetes.Clientset, err error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return client, nil
}
