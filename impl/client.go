package impl

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func K8SClient() (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", "") //使用InClusterConfig
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}
