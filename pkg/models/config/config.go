package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pkg/apis/config/dnsalias"
	"github.com/equinor/radix-operator/pkg/apis/defaults"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/equinor/radix-operator/pkg/apis/utils/git"
	tektonDefaults "github.com/equinor/radix-tekton/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type cfg struct {
	dnsConfig *dnsalias.DNSConfig
}

func (c *cfg) GetSourceDeploymentGitCommitHash() string {
	return viper.GetString(defaults.RadixPromoteSourceDeploymentCommitHashEnvironmentVariable)
}

func (c *cfg) GetSourceDeploymentGitBranch() string {
	return viper.GetString(defaults.RadixPromoteSourceDeploymentBranchEnvironmentVariable)
}

func (c *cfg) GetGitConfigMapName() string {
	return viper.GetString(defaults.RadixGitConfigMapEnvironmentVariable)
}

func (c *cfg) GetWebhookCommitId() string {
	return viper.GetString(defaults.RadixGithubWebhookCommitId)
}

// GetAppNamespace Radix application app-namespace
func (c *cfg) GetAppNamespace() string {
	return utils.GetAppNamespace(viper.GetString(defaults.RadixAppEnvironmentVariable))
}

// GetAppName Radix application name
func (c *cfg) GetAppName() string {
	return viper.GetString(defaults.RadixAppEnvironmentVariable)
}

// GetRadixConfigMapName Name of a ConfigMap, where Radix config file will be saved during RadixPipelineActionPrepare action
func (c *cfg) GetRadixConfigMapName() string {
	return viper.GetString(defaults.RadixConfigConfigMapEnvironmentVariable)
}

// GetRadixConfigBranch Name of the Radix application config branch
func (c *cfg) GetRadixConfigBranch() string {
	return viper.GetString(defaults.RadixConfigBranchEnvironmentVariable)
}

// GetRadixConfigFileName Name with path to the cloned Radix config file to be saved to a ConfigMap
func (c *cfg) GetRadixConfigFileName() string {
	return viper.GetString(defaults.RadixConfigFileEnvironmentVariable)
}

// GetRadixPipelineType Type of the pipeline, value of the radixv1.RadixPipelineType
func (c *cfg) GetRadixPipelineType() v1.RadixPipelineType {
	return v1.RadixPipelineType(viper.GetString(defaults.RadixPipelineTypeEnvironmentVariable))
}

// GetRadixImageTag Image tag for the built component
func (c *cfg) GetRadixImageTag() string {
	return viper.GetString(defaults.RadixImageTagEnvironmentVariable)
}

// GetBranch Branch of the Radix application to process in a pipeline
func (c *cfg) GetBranch() string {
	return viper.GetString(defaults.RadixBranchEnvironmentVariable)
}

// GetPipelinesAction  Pipeline action, one of values RadixPipelineActionPrepare, RadixPipelineActionRun
func (c *cfg) GetPipelinesAction() string {
	return viper.GetString(defaults.RadixPipelineActionEnvironmentVariable)
}

// GetRadixPipelineJobName Radix pipeline job name
func (c *cfg) GetRadixPipelineJobName() string {
	return viper.GetString(defaults.RadixPipelineJobEnvironmentVariable)
}

// GetRadixPromoteDeployment Radix pipeline promote deployment name
func (c *cfg) GetRadixPromoteDeployment() string {
	return viper.GetString(defaults.RadixPromoteDeploymentEnvironmentVariable)
}

// GetRadixPromoteFromEnvironment Radix pipeline promote deployment source environment name
func (c *cfg) GetRadixPromoteFromEnvironment() string {
	return viper.GetString(defaults.RadixPromoteFromEnvironmentEnvironmentVariable)
}

// GetRadixDeployToEnvironment Radix pipeline promote or deploy deployment target environment name
func (c *cfg) GetRadixDeployToEnvironment() string {
	return viper.GetString(defaults.RadixPromoteToEnvironmentEnvironmentVariable)
}

// GetDNSConfig The list of DNS aliases, reserved for Radix platform Radix application and services
func (c *cfg) GetDNSConfig() *dnsalias.DNSConfig {
	return c.dnsConfig
}

func validateReservedDNSAliases(dnsConfig *dnsalias.DNSConfig) {
	var errs []error
	if len(dnsConfig.ReservedAppDNSAliases) == 0 {
		errs = append(errs, fmt.Errorf("missing DNS aliases, reserved for Radix platform Radix application"))
	}
	if len(dnsConfig.ReservedDNSAliases) == 0 {
		errs = append(errs, fmt.Errorf("missing DNS aliases, reserved for Radix platform services"))
	}
	err := errors.Join(errs...)
	if err != nil {
		panic(err)
	}
}

// GetLogLevel Log level: ERROR, INFO (default), DEBUG
func (c *cfg) GetLogLevel() log.Level {
	switch viper.GetString(defaults.LogLevel) {
	case "DEBUG":
		return log.DebugLevel
	case "ERROR":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// GetGitRepositoryWorkspace Path to the cloned GitHub repository
func (c *cfg) GetGitRepositoryWorkspace() string {
	workspace := viper.GetString(tektonDefaults.RadixGithubWorkspaceEnvironmentVariable)
	if len(workspace) == 0 {
		return git.Workspace
	}
	return workspace
}

// Config for the pipeline
type Config interface {
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
	GetLogLevel() log.Level
	GetGitRepositoryWorkspace() string
	GetSourceDeploymentGitCommitHash() string
	GetSourceDeploymentGitBranch() string
	GetDNSConfig() *dnsalias.DNSConfig
}

// NewConfig New instance of a Config for the pipeline
func NewConfig() Config {
	viper.AutomaticEnv()
	dnsConfig := &dnsalias.DNSConfig{
		ReservedAppDNSAliases: maps.FromString(viper.GetString(defaults.RadixReservedAppDNSAliasesEnvironmentVariable)),
		ReservedDNSAliases:    strings.Split(viper.GetString(defaults.RadixReservedDNSAliasesEnvironmentVariable), ","),
	}
	validateReservedDNSAliases(dnsConfig)
	return &cfg{dnsConfig: dnsConfig}
}
