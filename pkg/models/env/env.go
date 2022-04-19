package env

import (
	"github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/spf13/viper"
)

type env struct {
}

func (e *env) GetAppNamespace() string {
	return utils.GetAppNamespace(viper.GetString(defaults.RadixAppEnvironmentVariable))
}

func (e *env) GetAppName() string {
	return viper.GetString(defaults.RadixAppEnvironmentVariable)
}

func (e *env) GetConfigMapName() string {
	return viper.GetString(defaults.RadixConfigConfigMapEnvironmentVariable)
}

func (e *env) GetRadixConfigFileName() string {
	return viper.GetString(defaults.RadixConfigFileEnvironmentVariable)
}

func (e *env) GetRadixPipelineType() string {
	return viper.GetString(defaults.RadixPipelineTypeEnvironmentVariable)
}

func (e *env) GetRadixPipelineRun() string {
	return viper.GetString(defaults.RadixPipelineRunEnvironmentVariable)
}

func (e *env) GetRadixImageTag() string {
	return viper.GetString(defaults.RadixImageTagEnvironmentVariable)
}

func (e *env) GetBranch() string {
	return viper.GetString(defaults.RadixBranchEnvironmentVariable)
}

func (e *env) GetTektonAction() string {
	return viper.GetString(defaults.RadixTektonActionEnvironmentVariable)
}

func (e *env) GetRadixPipelineJobName() string {
	return viper.GetString(defaults.RadixPipelineJobEnvironmentVariable)
}

type Env interface {
	GetAppName() string
	GetAppNamespace() string
	GetRadixPipelineJobName() string
	GetConfigMapName() string
	GetRadixConfigFileName() string
	GetRadixPipelineType() string
	GetRadixPipelineRun() string
	GetRadixImageTag() string
	GetBranch() string
	GetTektonAction() string
}

func NewEnvironment() Env {
	viper.AutomaticEnv()
	return &env{}
}
