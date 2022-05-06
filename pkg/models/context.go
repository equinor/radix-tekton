package models

import (
	"github.com/equinor/radix-tekton/pkg/models/env"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

//Context of the pipeline
type Context interface {
	ProcessRadixAppConfig() error
	RunTektonPipelineJob() error
	GetEnv() env.Env
	GetHash() string
	GetKubeClient() kubernetes.Interface
	GetTektonClient() tektonclient.Interface
}
