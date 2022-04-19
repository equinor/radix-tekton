package pipeline

import (
	"github.com/equinor/radix-operator/pipeline-runner/steps"
	"github.com/equinor/radix-operator/pkg/apis/applicationconfig"
	"github.com/equinor/radix-tekton/pkg/configmap"
	log "github.com/sirupsen/logrus"
	"strings"
)

//ProcessRadixAppConfig Load radixconfig.yaml to a ConfigMap and create RadixApplication
func (ctx *pipelineContext) ProcessRadixAppConfig() error {
	configFileContent, err := configmap.CreateFromFile(ctx.kubeClient, ctx.env)
	if err != nil {
		log.Fatalf("Error copying radixconfig.yaml and creating config map from file: %v", err)
	}
	log.Debugln("radixconfig.yaml has been loaded")

	ctx.radixApplication, err = steps.CreateRadixApplication(ctx.radixClient, configFileContent)
	if err != nil {
		return err
	}
	log.Debugln("Radix Application has been loaded")

	branchIsMapped, targetEnvironments := applicationconfig.IsThereAnythingToDeployForRadixApplication(ctx.env.
		GetBranch(), ctx.radixApplication)
	if !branchIsMapped {
		log.Infof("No environments are mapped to the branch '%s'.", ctx.env.GetBranch())
		return nil
	}

	log.Infof("Environment(s) %v are mapped to the branch '%s'.", getEnvironmentList(targetEnvironments), ctx.env.GetBranch())
	ctx.targetEnvironments = targetEnvironments

	err = ctx.prepareTektonPipelineJob()
	if err != nil {
		return err
	}
	log.Debugln("Tekton pipelines have been loaded")

	return nil
}

func getEnvironmentList(environmentNameMap map[string]bool) string {
	var envNames []string
	for envName := range environmentNameMap {
		envNames = append(envNames, envName)
	}
	return strings.Join(envNames, ", ")
}
