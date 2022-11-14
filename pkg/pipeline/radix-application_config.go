package pipeline

import (
	"context"
	"fmt"
	"strings"

	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-operator/pipeline-runner/model"
	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-operator/pkg/apis/applicationconfig"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//ProcessRadixAppConfig Load Radix config file to a ConfigMap and create RadixApplication
func (ctx *pipelineContext) ProcessRadixAppConfig() error {
	configFileContent, err := configmap.CreateFromRadixConfigFile(ctx.env)
	if err != nil {
		log.Fatalf("Error copying Radix config file %s and creating config map from it: %v", ctx.GetEnv().GetRadixConfigFileName(), err)
	}
	log.Debugln(fmt.Sprintf("Radix config file %s has been loaded", ctx.GetEnv().GetRadixConfigFileName()))

	ctx.radixApplication, err = ctx.createRadixApplicationFromContent(configFileContent)
	if err != nil {
		return err
	}
	log.Debugln("Radix Application has been loaded")

	envIsValid, err := ctx.setTargetEnvironments()
	if err != nil || !envIsValid {
		return err
	}

	log.Debugln("Target environments have been loaded")

	prepareBuildContext, err := ctx.preparePipelinesJob()
	if err != nil {
		return err
	}
	return ctx.createConfigMap(configFileContent, prepareBuildContext)
}

func (ctx *pipelineContext) createConfigMap(configFileContent string, prepareBuildContext *model.PrepareBuildContext) error {
	env := ctx.GetEnv()
	buildContext, err := yaml.Marshal(prepareBuildContext)
	if err != nil {
		return err
	}
	_, err = ctx.kubeClient.CoreV1().ConfigMaps(env.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.GetRadixConfigMapName(),
				Namespace: env.GetAppNamespace(),
				Labels:    map[string]string{kube.RadixJobNameLabel: ctx.GetEnv().GetRadixPipelineJobName()},
			},
			Data: map[string]string{
				pipelineDefaults.PipelineConfigMapContent:      configFileContent,
				pipelineDefaults.PipelineConfigMapBuildContext: string(buildContext),
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Created ConfigMap %s", env.GetRadixConfigMapName())
	return nil
}

func (ctx *pipelineContext) setTargetEnvironments() (bool, error) {
	if ctx.GetEnv().GetRadixPipelineType() == v1.Promote {
		return true, ctx.setTargetEnvironmentsForPromote()
	}
	if ctx.GetEnv().GetRadixPipelineType() == v1.Deploy {
		return true, ctx.setTargetEnvironmentsForDeploy()
	}
	branchIsMapped, targetEnvironments := applicationconfig.IsThereAnythingToDeployForRadixApplication(ctx.env.GetBranch(), ctx.radixApplication)
	if !branchIsMapped {
		log.Infof("no environments are mapped to the branch %s", ctx.env.GetBranch())
		return false, nil
	}
	ctx.targetEnvironments = make(map[string]bool)
	for envName, isEnvTarget := range targetEnvironments {
		if isEnvTarget { //get only target environments
			ctx.targetEnvironments[envName] = true
		}
	}
	log.Infof("Environment(s) %v are mapped to the branch %s.", getEnvironmentList(ctx.targetEnvironments), ctx.env.GetBranch())
	log.Infof("pipeline type: %s", ctx.env.GetRadixPipelineType())
	return true, nil
}

func (ctx *pipelineContext) setTargetEnvironmentsForPromote() error {
	var errs []error
	if len(ctx.env.GetRadixPromoteDeployment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote deployment name"))
	}
	if len(ctx.env.GetRadixPromoteFromEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote source environment name"))
	}
	if len(ctx.env.GetRadixDeployToEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote target environment name"))
	}
	if len(errs) > 0 {
		log.Infoln("pipeline type: promote")
		return commonErrors.Concat(errs)
	}
	ctx.targetEnvironments = map[string]bool{ctx.env.GetRadixDeployToEnvironment(): true} //run Tekton pipelines for the promote target environment
	log.Infof("promote the deployment %s from the environment %s to %s", ctx.env.GetRadixPromoteDeployment(), ctx.env.GetRadixPromoteFromEnvironment(), ctx.env.GetRadixDeployToEnvironment())
	return nil
}

func (ctx *pipelineContext) setTargetEnvironmentsForDeploy() error {
	targetEnvironment := ctx.env.GetRadixDeployToEnvironment()
	if len(targetEnvironment) == 0 {
		return fmt.Errorf("no target environment is specified for the deploy pipeline")
	}
	ctx.targetEnvironments = map[string]bool{targetEnvironment: true}
	log.Infof("Target environment: %v", targetEnvironment)
	log.Infof("pipeline type: %s", ctx.env.GetRadixPipelineType())
	return nil
}

func getEnvironmentList(environmentNameMap map[string]bool) string {
	var envNames []string
	for envName := range environmentNameMap {
		envNames = append(envNames, envName)
	}
	return strings.Join(envNames, ", ")
}
