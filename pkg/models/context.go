package models

import (
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/internal/wait"
	"github.com/equinor/radix-tekton/pkg/models/env"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// Context of the pipeline
type Context interface {
	// ProcessRadixAppConfig Load Radix config file to a ConfigMap and create RadixApplication
	ProcessRadixAppConfig() error
	// RunPipelinesJob un the job, which creates Tekton PipelineRun-s
	RunPipelinesJob() error
	// GetEnv Environment for the pipeline
	GetEnv() env.Env
	// GetHash Hash, common for all pipeline Kubernetes object names
	GetHash() string
	// GetKubeClient Kubernetes client
	GetKubeClient() kubernetes.Interface
	// GetTektonClient Tekton client
	GetTektonClient() tektonclient.Interface
	// GetRadixApplication Gets the RadixApplication, loaded from the config-map
	GetRadixApplication() *v1.RadixApplication
	// GetPipelineRunsWaiter Returns a waiter that returns when all pipelineruns have completed
	GetPipelineRunsWaiter() wait.PipelineRunsCompletionWaiter
	// WithPipelineRunsWaiter allows to change the current waiter for a adifferent implementation
	WithPipelineRunsWaiter(waiter wait.PipelineRunsCompletionWaiter)
}
