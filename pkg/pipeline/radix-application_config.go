package pipeline

import (
	"fmt"
	"strings"

	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-operator/pkg/apis/applicationconfig"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
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
	if ctx.GetEnv().GetRadixPipelineType() != v1.Promote {
		branchIsMapped, targetEnvironments := applicationconfig.IsThereAnythingToDeployForRadixApplication(ctx.env.GetBranch(), ctx.radixApplication)
		if !branchIsMapped {
			return fmt.Errorf("no environments are mapped to the branch %s", ctx.env.GetBranch())
		}
		ctx.targetEnvironments = make(map[string]bool)
		for envName, isEnvTarget := range targetEnvironments {
			if isEnvTarget { //get only target environments
				ctx.targetEnvironments[envName] = true
			}
		}
		log.Infof("Environment(s) %v are mapped to the branch %s.", getEnvironmentList(ctx.targetEnvironments), ctx.env.GetBranch())
		log.Infof("pipeline type: %s", ctx.env.GetRadixPipelineType())
		return nil
	}
	var errs []error
	if len(ctx.env.GetRadixPromoteDeployment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote deployment name"))
	}
	if len(ctx.env.GetRadixPromoteFromEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote source environment name"))
	}
	if len(ctx.env.GetRadixPromoteToEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote target environment name"))
	}
	if len(errs) > 0 {
		log.Infoln("pipeline type: promote")
		return commonErrors.Concat(errs)
	}
	ctx.targetEnvironments = map[string]bool{ctx.env.GetRadixPromoteToEnvironment(): true} //run Tekton pipelines for the promote target environment
	log.Infof("promote the deployment %s from the environment %s to %s", ctx.env.GetRadixPromoteDeployment(), ctx.env.GetRadixPromoteFromEnvironment(), ctx.env.GetRadixPromoteToEnvironment())
	return nil
}

func getEnvironmentList(environmentNameMap map[string]bool) string {
	var envNames []string
	for envName := range environmentNameMap {
		envNames = append(envNames, envName)
	}
	return strings.Join(envNames, ", ")
}
