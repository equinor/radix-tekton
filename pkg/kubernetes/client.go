package kubernetes

import (
	"fmt"
	"os"

	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClients Gets clients to talk to the API
func GetClients() (kubernetes.Interface, radixclient.Interface, tektonclient.Interface, error) {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("getClusterConfig InClusterConfig: %v", err)
		}
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getClusterConfig k8s kubeClient: %v", err)
	}

	radixClient, err := radixclient.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getClusterConfig radix kubeClient: %v", err)
	}

	tektonClient, err := tektonclient.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("getClusterConfig radix tektonClient: %v", err)
	}

	return kubeClient, radixClient, tektonClient, err
}
