package test

import (
	"context"

	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	radixclientfake "github.com/equinor/radix-operator/pkg/client/clientset/versioned/fake"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonclientfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeclientfake "k8s.io/client-go/kubernetes/fake"
)

func Setup() (kubernetes.Interface, radixclient.Interface, tektonclient.Interface) {
	kubeclient := kubeclientfake.NewSimpleClientset()
	rxclient := radixclientfake.NewSimpleClientset()
	tknclient := tektonclientfake.NewSimpleClientset()
	return kubeclient, rxclient, tknclient
}

func (params *TestParams) ApplyRd(radixClient radixclient.Interface) (*v1.RadixDeployment, error) {
	rd := utils.ARadixDeployment().
		WithDeploymentName(params.DeploymentName).
		WithAppName(params.AppName).
		WithEnvironment(params.Environment).
		WithComponents().
		WithJobComponents().
		BuildRD()
	_, err := radixClient.RadixV1().RadixDeployments(rd.Namespace).Create(context.Background(), rd, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return rd, nil
}

type TestParams struct {
	AppName          string
	Environment      string
	Namespace        string
	JobComponentName string
	DeploymentName   string
}
