package kubernetes

import (
	"context"
	"fmt"
	"os"
	"time"

	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClients Gets clients to talk to the API
func GetClients() (kubernetes.Interface, radixclient.Interface, tektonclient.Interface, error) {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	ctx := context.Background()

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("getClusterConfig InClusterConfig: %v", err)
		}
	}

	kubeClient, err := waitForClientWithSuccessfulConnection(ctx, func() (*kubernetes.Clientset, error) {
		return kubernetes.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Kubernetes resource client: %w", err)
	}

	radixClient, err := waitForClientWithSuccessfulConnection(ctx, func() (*radixclient.Clientset, error) {
		return radixclient.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Radix resource client: %w", err)
	}

	tektonClient, err := waitForClientWithSuccessfulConnection(ctx, func() (*tektonclient.Clientset, error) {
		return tektonclient.NewForConfig(config)
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Tekton resource client: %w", err)
	}

	return kubeClient, radixClient, tektonClient, err
}

func waitForClientWithSuccessfulConnection[T interface{ RESTClient() rest.Interface }](ctx context.Context, clientFactory func() (T, error)) (T, error) {
	var client T
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err := wait.PollUntilContextCancel(timeoutCtx, 15*time.Second, true, func(ctx context.Context) (done bool, err error) {
		c, err := clientFactory()
		if err != nil {
			return false, err
		}

		// Retry if error transient, e.g. TLS handshake timeout
		if err := c.RESTClient().Get().Do(timeoutCtx).Error(); err != nil && isTransientConnectionError(err) {
			log.Infof("transient error when connecting, retrying: %v", err)
			return false, nil
		}
		client = c
		return true, nil
	})
	return client, err
}

func isTransientConnectionError(err error) bool {
	// TODO: Should we check for other connection errors that are transient, e.g. net.DNSError?
	return err.Error() == "net/http: TLS handshake timeout"
}
