package pipeline

import (
	"fmt"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
	"strings"

	"github.com/equinor/radix-operator/pkg/apis/applicationconfig"
	log "github.com/sirupsen/logrus"
)

//ProcessRadixAppConfig Load radixconfig.yaml to a ConfigMap and create RadixApplication
func (ctx *pipelineContext) ProcessRadixAppConfig() error {
	configFileContent, err := configmap.CreateFromRadixConfigFile(ctx.kubeClient, ctx.env)
	if err != nil {
		log.Fatalf("Error copying radixconfig.yaml and creating config map from file: %v", err)
	}
	log.Debugln("radixconfig.yaml has been loaded")

	ctx.radixApplication, err = ctx.createRadixApplicationFromContent(configFileContent)
	if err != nil {
		return err
	}
	log.Debugln("Radix Application has been loaded")

	err = ctx.setTargetEnvironments()
	if err != nil {
		return err
	}

	log.Debugln("Target environments have been loaded")

	return ctx.preparePipelinesJob()
}

func (ctx *pipelineContext) setTargetEnvironments() error {
	branchIsMapped, targetEnvironments := applicationconfig.IsThereAnythingToDeployForRadixApplication(ctx.env.GetBranch(), ctx.radixApplication)
	if !branchIsMapped {
		return fmt.Errorf("no environments are mapped to the branch '%s'", ctx.env.GetBranch())
	}
	ctx.targetEnvironments = make(map[string]bool)
	for envName, isEnvTarget := range targetEnvironments {
		if isEnvTarget { //get only target environments
			ctx.targetEnvironments[envName] = true
		}
	}
	log.Infof("Environment(s) %v are mapped to the branch '%s'.", getEnvironmentList(ctx.targetEnvironments), ctx.env.GetBranch())
	return nil
}

func getEnvironmentList(environmentNameMap map[string]bool) string {
	var envNames []string
	for envName := range environmentNameMap {
		envNames = append(envNames, envName)
	}
	return strings.Join(envNames, ", ")
}
