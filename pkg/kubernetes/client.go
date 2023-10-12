package kubernetes

import (
	"context"
	"fmt"
	"os"
	"time"

	operatorutils "github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClients Gets clients to talk to the API
func GetClients() (kubernetes.Interface, radixclient.Interface, tektonclient.Interface, error) {
	pollTimeout, pollInterval := time.Minute, 15*time.Second
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	ctx := context.Background()

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("getClusterConfig InClusterConfig: %v", err)
		}
	}

	kubeClient, err := operatorutils.PollUntilRESTClientSuccessfulConnection(ctx, pollTimeout, pollInterval, func() (*kubernetes.Clientset, error) {
		return kubernetes.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Kubernetes resource client: %w", err)
	}

	radixClient, err := operatorutils.PollUntilRESTClientSuccessfulConnection(ctx, pollTimeout, pollInterval, func() (*radixclient.Clientset, error) {
		return radixclient.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Radix resource client: %w", err)
	}

	tektonClient, err := operatorutils.PollUntilRESTClientSuccessfulConnection(ctx, pollTimeout, pollInterval, func() (*tektonclient.Clientset, error) {
		return tektonclient.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Tekton resource client: %w", err)
	}

	return kubeClient, radixClient, tektonClient, err
}
