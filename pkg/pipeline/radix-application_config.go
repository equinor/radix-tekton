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
	radixv1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// ProcessRadixAppConfig Load Radix config file to a ConfigMap and create RadixApplication
func (ctx *pipelineContext) ProcessRadixAppConfig() error {
	configFileContent, err := configmap.CreateFromRadixConfigFile(ctx.cfg)
	if err != nil {
		return fmt.Errorf("error reading the Radix config file %s: %v", ctx.GetConfig().GetRadixConfigFileName(), err)
	}
	log.Debugf("Radix config file %s has been loaded", ctx.GetConfig().GetRadixConfigFileName())

	ctx.radixApplication, err = ctx.createRadixApplicationFromContent(configFileContent)
	if err != nil {
		return err
	}
	log.Debug("Radix Application has been loaded")

	err = ctx.setTargetEnvironments()
	if err != nil {
		return err
	}
	prepareBuildContext, err := ctx.preparePipelinesJob()
	if err != nil {
		return err
	}
	return ctx.createConfigMap(configFileContent, prepareBuildContext)
}

func (ctx *pipelineContext) createConfigMap(configFileContent string, prepareBuildContext *model.PrepareBuildContext) error {
	cfg := ctx.GetConfig()
	if prepareBuildContext == nil {
		prepareBuildContext = &model.PrepareBuildContext{}
	}
	buildContext, err := yaml.Marshal(prepareBuildContext)
	if err != nil {
		return err
	}
	_, err = ctx.kubeClient.CoreV1().ConfigMaps(cfg.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.GetRadixConfigMapName(),
				Namespace: cfg.GetAppNamespace(),
				Labels:    map[string]string{kube.RadixJobNameLabel: ctx.GetConfig().GetRadixPipelineJobName()},
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
	log.Debugf("Created ConfigMap %s", cfg.GetRadixConfigMapName())
	return nil
}

func (ctx *pipelineContext) setTargetEnvironments() error {
	log.Debug("Set target environment")
	if ctx.GetConfig().GetRadixPipelineType() == radixv1.Promote {
		return ctx.setTargetEnvironmentsForPromote()
	}
	if ctx.GetConfig().GetRadixPipelineType() == radixv1.Deploy {
		return ctx.setTargetEnvironmentsForDeploy()
	}
	targetEnvironments := applicationconfig.GetTargetEnvironments(ctx.cfg.GetBranch(), ctx.radixApplication)
	ctx.targetEnvironments = make(map[string]bool)
	for _, envName := range targetEnvironments {
		ctx.targetEnvironments[envName] = true
	}
	if len(ctx.targetEnvironments) > 0 {
		log.Infof("Environment(s) %v are mapped to the branch %s.", getEnvironmentList(ctx.targetEnvironments), ctx.cfg.GetBranch())
	} else {
		log.Infof("No environments are mapped to the branch %s.", ctx.cfg.GetBranch())
	}
	log.Infof("pipeline type: %s", ctx.cfg.GetRadixPipelineType())
	return nil
}

func (ctx *pipelineContext) setTargetEnvironmentsForPromote() error {
	var errs []error
	if len(ctx.cfg.GetRadixPromoteDeployment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote deployment name"))
	}
	if len(ctx.cfg.GetRadixPromoteFromEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote source environment name"))
	}
	if len(ctx.cfg.GetRadixDeployToEnvironment()) == 0 {
		errs = append(errs, fmt.Errorf("missing promote target environment name"))
	}
	if len(errs) > 0 {
		log.Infoln("pipeline type: promote")
		return commonErrors.Concat(errs)
	}
	ctx.targetEnvironments = map[string]bool{ctx.cfg.GetRadixDeployToEnvironment(): true} // run Tekton pipelines for the promote target environment
	log.Infof("promote the deployment %s from the environment %s to %s", ctx.cfg.GetRadixPromoteDeployment(), ctx.cfg.GetRadixPromoteFromEnvironment(), ctx.cfg.GetRadixDeployToEnvironment())
	return nil
}

func (ctx *pipelineContext) setTargetEnvironmentsForDeploy() error {
	targetEnvironment := ctx.cfg.GetRadixDeployToEnvironment()
	if len(targetEnvironment) == 0 {
		return fmt.Errorf("no target environment is specified for the deploy pipeline")
	}
	ctx.targetEnvironments = map[string]bool{targetEnvironment: true}
	log.Infof("Target environment: %v", targetEnvironment)
	log.Infof("pipeline type: %s", ctx.cfg.GetRadixPipelineType())
	return nil
}

func getEnvironmentList(environmentNameMap map[string]bool) string {
	var envNames []string
	for envName := range environmentNameMap {
		envNames = append(envNames, envName)
	}
	return strings.Join(envNames, ", ")
}
