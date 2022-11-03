package test

import (
	"context"
	"testing"

	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	radixclientfake "github.com/equinor/radix-operator/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeclientfake "k8s.io/client-go/kubernetes/fake"
)

func Setup(t *testing.T) (radixclient.Interface, kubernetes.Interface) {
	kubeclient := kubeclientfake.NewSimpleClientset()
	radixclient := radixclientfake.NewSimpleClientset()
	return radixclient, kubeclient
}

func (params *TestParams) ApplyRd(radixClient radixclient.Interface) *v1.RadixDeployment {
	rd := utils.ARadixDeployment().
		WithDeploymentName(params.DeploymentName).
		WithAppName(params.AppName).
		WithEnvironment(params.Environment).
		WithComponents().
		WithJobComponents().
		BuildRD()
	radixClient.RadixV1().RadixDeployments(rd.Namespace).Create(context.Background(), rd, metav1.CreateOptions{})
	return rd
}

type TestParams struct {
	AppName          string
	Environment      string
	Namespace        string
	JobComponentName string
	DeploymentName   string
}