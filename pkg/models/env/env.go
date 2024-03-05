package env

import (
	"strings"

	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pkg/apis/config/dnsalias"
	"github.com/equinor/radix-operator/pkg/apis/defaults"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/equinor/radix-operator/pkg/apis/utils/git"
	tektonDefaults "github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/models/env/internal"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type env struct {
	dnsConfig *dnsalias.DNSConfig
}

func (e *env) GetSourceDeploymentGitCommitHash() string {
	return viper.GetString(defaults.RadixPromoteSourceDeploymentCommitHashEnvironmentVariable)
}

func (e *env) GetSourceDeploymentGitBranch() string {
	return viper.GetString(defaults.RadixPromoteSourceDeploymentBranchEnvironmentVariable)
}

func (e *env) GetGitConfigMapName() string {
	return viper.GetString(defaults.RadixGitConfigMapEnvironmentVariable)
}

func (e *env) GetWebhookCommitId() string {
	return viper.GetString(defaults.RadixGithubWebhookCommitId)
}

// GetAppNamespace Radix application app-namespace
func (e *env) GetAppNamespace() string {
	return utils.GetAppNamespace(viper.GetString(defaults.RadixAppEnvironmentVariable))
}

// GetAppName Radix application name
func (e *env) GetAppName() string {
	return viper.GetString(defaults.RadixAppEnvironmentVariable)
}

// GetRadixConfigMapName Name of a ConfigMap, where Radix config file will be saved during RadixPipelineActionPrepare action
func (e *env) GetRadixConfigMapName() string {
	return viper.GetString(defaults.RadixConfigConfigMapEnvironmentVariable)
}

// GetRadixConfigBranch Name of the Radix application config branch
func (e *env) GetRadixConfigBranch() string {
	return viper.GetString(defaults.RadixConfigBranchEnvironmentVariable)
}

// GetRadixConfigFileName Name with path to the cloned Radix config file to be saved to a ConfigMap
func (e *env) GetRadixConfigFileName() string {
	return viper.GetString(defaults.RadixConfigFileEnvironmentVariable)
}

// GetRadixPipelineType Type of the pipeline, value of the radixv1.RadixPipelineType
func (e *env) GetRadixPipelineType() v1.RadixPipelineType {
	return v1.RadixPipelineType(viper.GetString(defaults.RadixPipelineTypeEnvironmentVariable))
}

// GetRadixImageTag Image tag for the built component
func (e *env) GetRadixImageTag() string {
	return viper.GetString(defaults.RadixImageTagEnvironmentVariable)
}

// GetBranch Branch of the Radix application to process in a pipeline
func (e *env) GetBranch() string {
	return viper.GetString(defaults.RadixBranchEnvironmentVariable)
}

// GetPipelinesAction  Pipeline action, one of values RadixPipelineActionPrepare, RadixPipelineActionRun
func (e *env) GetPipelinesAction() string {
	return viper.GetString(defaults.RadixPipelineActionEnvironmentVariable)
}

// GetRadixPipelineJobName Radix pipeline job name
func (e *env) GetRadixPipelineJobName() string {
	return viper.GetString(defaults.RadixPipelineJobEnvironmentVariable)
}

// GetRadixPromoteDeployment Radix pipeline promote deployment name
func (e *env) GetRadixPromoteDeployment() string {
	return viper.GetString(defaults.RadixPromoteDeploymentEnvironmentVariable)
}

// GetRadixPromoteFromEnvironment Radix pipeline promote deployment source environment name
func (e *env) GetRadixPromoteFromEnvironment() string {
	return viper.GetString(defaults.RadixPromoteFromEnvironmentEnvironmentVariable)
}

// GetRadixDeployToEnvironment Radix pipeline promote or deploy deployment target environment name
func (e *env) GetRadixDeployToEnvironment() string {
	return viper.GetString(defaults.RadixPromoteToEnvironmentEnvironmentVariable)
}

// GetDNSConfig The list of DNS aliases, reserved for Radix platform Radix application and services
func (e *env) GetDNSConfig() *dnsalias.DNSConfig {
	return e.dnsConfig
}

// GetLogLevel Log level: ERROR, INFO (default), DEBUG
func (e *env) GetLogLevel() (zerolog.Level, error) {
	level := viper.GetString(defaults.LogLevel)
	if level == "" {
		return zerolog.InfoLevel, nil
	}

	return zerolog.ParseLevel(level)
}

// GetPrettyPrint Format colored output instead of (default) JSON
func (e *env) GetPrettyPrint() bool {
	return viper.GetBool("PRETTY_PRINT")
}

// GetGitRepositoryWorkspace Path to the cloned GitHub repository
func (e *env) GetGitRepositoryWorkspace() string {
	workspace := viper.GetString(tektonDefaults.RadixGithubWorkspaceEnvironmentVariable)
	if len(workspace) == 0 {
		return git.Workspace
	}
	return workspace
}

// Env Environment for the pipeline
type Env interface {
	GetAppName() string
	GetAppNamespace() string
	GetRadixPipelineJobName() string
	GetRadixConfigMapName() string
	GetGitConfigMapName() string
	GetWebhookCommitId() string
	GetRadixConfigBranch() string
	GetRadixConfigFileName() string
	GetRadixPipelineType() v1.RadixPipelineType
	GetRadixPromoteDeployment() string
	GetRadixPromoteFromEnvironment() string
	GetRadixDeployToEnvironment() string
	GetRadixImageTag() string
	GetBranch() string
	GetPipelinesAction() string
	GetLogLevel() (zerolog.Level, error)
	GetPrettyPrint() bool
	GetGitRepositoryWorkspace() string
	GetSourceDeploymentGitCommitHash() string
	GetSourceDeploymentGitBranch() string
	GetDNSConfig() *dnsalias.DNSConfig
}

// NewEnvironment New instance of an Environment for the pipeline
func NewEnvironment() Env {
	viper.AutomaticEnv()
	dnsConfig := &dnsalias.DNSConfig{
		ReservedAppDNSAliases: maps.FromString(viper.GetString(defaults.RadixReservedAppDNSAliasesEnvironmentVariable)),
		ReservedDNSAliases:    strings.Split(viper.GetString(defaults.RadixReservedDNSAliasesEnvironmentVariable), ","),
	}
	if err := internal.ValidateReservedDNSAliases(dnsConfig); err != nil {
		panic(err)
	}
	return &env{dnsConfig: dnsConfig}
}
