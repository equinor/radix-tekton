package env

import (
	"github.com/equinor/radix-operator/pkg/apis/defaults"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type env struct {
}

//GetAppNamespace Radix application app-namespace
func (e *env) GetAppNamespace() string {
	return utils.GetAppNamespace(viper.GetString(defaults.RadixAppEnvironmentVariable))
}

//GetAppName Radix application name
func (e *env) GetAppName() string {
	return viper.GetString(defaults.RadixAppEnvironmentVariable)
}

//GetConfigMapName Name of a ConfigMap, where radixconfig.yaml will be saved during RadixPipelineActionPrepare action
func (e *env) GetConfigMapName() string {
	return viper.GetString(defaults.RadixConfigConfigMapEnvironmentVariable)
}

//GetRadixConfigFileName Name with path to the cloned radixconfig.yaml file to be saved to a ConfigMap
func (e *env) GetRadixConfigFileName() string {
	return viper.GetString(defaults.RadixConfigFileEnvironmentVariable)
}

//GetRadixPipelineType Type of the pipeline, value of the radixv1.RadixPipelineType
func (e *env) GetRadixPipelineType() v1.RadixPipelineType {
	return v1.RadixPipelineType(viper.GetString(defaults.RadixPipelineTypeEnvironmentVariable))
}

//GetRadixImageTag Image tag for the built component
func (e *env) GetRadixImageTag() string {
	return viper.GetString(defaults.RadixImageTagEnvironmentVariable)
}

//GetBranch Branch of the Radix application to process in a pipeline
func (e *env) GetBranch() string {
	return viper.GetString(defaults.RadixBranchEnvironmentVariable)
}

//GetPipelinesAction  Pipeline action, one of values RadixPipelineActionPrepare, RadixPipelineActionRun
func (e *env) GetPipelinesAction() string {
	return viper.GetString(defaults.RadixPipelineActionEnvironmentVariable)
}

//GetRadixPipelineJobName Radix pipeline job name
func (e *env) GetRadixPipelineJobName() string {
	return viper.GetString(defaults.RadixPipelineJobEnvironmentVariable)
}

//GetRadixPromoteDeployment Radix pipeline promote deployment name
func (e *env) GetRadixPromoteDeployment() string {
	return viper.GetString(defaults.RadixPromoteDeploymentEnvironmentVariable)
}

//GetRadixPromoteFromEnvironment Radix pipeline promote deployment source environment name
func (e *env) GetRadixPromoteFromEnvironment() string {
	return viper.GetString(defaults.RadixPromoteFromEnvironmentEnvironmentVariable)
}

//GetRadixPromoteToEnvironment Radix pipeline promote deployment target environment name
func (e *env) GetRadixPromoteToEnvironment() string {
	return viper.GetString(defaults.RadixPromoteToEnvironmentEnvironmentVariable)
}

//GetLogLevel Log level: ERROR, INFO (default), DEBUG
func (e *env) GetLogLevel() log.Level {
	switch viper.GetString(defaults.LogLevel) {
	case "DEBUG":
		return log.DebugLevel
	case "ERROR":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

//Env Environment for the pipeline
type Env interface {
	GetAppName() string
	GetAppNamespace() string
	GetRadixPipelineJobName() string
	GetConfigMapName() string
	GetRadixConfigFileName() string
	GetRadixPipelineType() v1.RadixPipelineType
	GetRadixPromoteDeployment() string
	GetRadixPromoteFromEnvironment() string
	GetRadixPromoteToEnvironment() string
	GetRadixImageTag() string
	GetBranch() string
	GetPipelinesAction() string
	GetLogLevel() log.Level
}

//NewEnvironment New instance of an Environment for the pipeline
func NewEnvironment() Env {
	viper.AutomaticEnv()
	return &env{}
}
