package models

import (
	"github.com/equinor/radix-tekton/pkg/models/env"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

//Context of the pipeline
type Context interface {
	//ProcessRadixAppConfig Load radixconfig.yaml to a ConfigMap and create RadixApplication
	ProcessRadixAppConfig() error
	//RunPipelinesJob un the job, which creates Tekton PipelineRun-s
	RunPipelinesJob() error
	//GetEnv Environment for the pipeline
	GetEnv() env.Env
	//GetHash Hash, common for all pipeline Kubernetes object names
	GetHash() string
	//GetKubeClient Kubernetes client
	GetKubeClient() kubernetes.Interface
	//GetTektonClient Tekton client
	GetTektonClient() tektonclient.Interface
}
