package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/utils/labels"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonInformerFactory "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	knativeApis "knative.dev/pkg/apis"
	knative "knative.dev/pkg/apis/duck/v1beta1"
)

//RunTektonPipelineJob Run the job, which creates Tekton PipelineRun-s for each preliminary prepared pipelines of the specified branch
func (ctx *pipelineContext) RunPipelinesJob() error {
	namespace := ctx.env.GetAppNamespace()
	pipelineList, err := ctx.tektonClient.TektonV1beta1().Pipelines(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", kube.RadixJobNameLabel, ctx.env.GetRadixPipelineJobName()),
	})
	if err != nil {
		return err
	}
	if len(pipelineList.Items) == 0 {
		log.Infof("no pipelines exist, skip Tekton pipeline run.")
		return nil
	}

	radixApplication, err := ctx.createRadixApplicationFromConfigMap()
	if err != nil {
		return err
	}
	ctx.radixApplication = radixApplication
	err = ctx.setTargetEnvironments()
	if err != nil {
		return err
	}

	log.Infof("Run tekton pipelines for the branch %s", ctx.env.GetBranch())

	pipelineRunMap, err := ctx.runPipelines(pipelineList.Items, namespace)

	if err != nil {
		err = fmt.Errorf("failed to run pipelines: %w", err)
		deleteErrors := ctx.deletePipelineRuns(pipelineRunMap, namespace)
		if len(deleteErrors) > 0 {
			deleteErrors = append(deleteErrors, err)
			return commonErrors.Concat(deleteErrors)
		}
		return err
	}

	err = ctx.WaitForCompletionOf(pipelineRunMap)
	if err != nil {
		return fmt.Errorf("failed tekton pipelines, %v, for app %s. %w",
			ctx.targetEnvironments, ctx.env.GetAppName(),
			err)
	}
	return nil
}

func (ctx *pipelineContext) deletePipelineRuns(pipelineRunMap map[string]*v1beta1.PipelineRun, namespace string) []error {
	var deleteErrors []error
	for _, pipelineRun := range pipelineRunMap {
		log.Debugf("delete the pipeline-run %s", pipelineRun.Name)
		deleteErr := ctx.tektonClient.TektonV1beta1().PipelineRuns(namespace).
			Delete(context.Background(), pipelineRun.GetName(), metav1.DeleteOptions{})
		if deleteErr != nil {
			log.Debugf("failed to delete the pipeline-run %s", pipelineRun.Name)
			deleteErrors = append(deleteErrors, deleteErr)
		}
	}
	return deleteErrors
}

func (ctx *pipelineContext) runPipelines(pipelines []v1beta1.Pipeline, namespace string) (map[string]*v1beta1.PipelineRun, error) {
	timestamp := time.Now().Format("20060102150405")
	pipelineRunMap := make(map[string]*v1beta1.PipelineRun)
	var errs []error
	for _, pipeline := range pipelines {
		createdPipelineRun, err := ctx.createPipelineRun(namespace, &pipeline, timestamp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		pipelineRunMap[createdPipelineRun.GetName()] = createdPipelineRun
	}
	if len(errs) > 0 {
		return pipelineRunMap, commonErrors.Concat(errs)
	}
	return pipelineRunMap, nil
}

func (ctx *pipelineContext) createPipelineRun(namespace string, pipeline *v1beta1.Pipeline, timestamp string) (*v1beta1.PipelineRun, error) {
	targetEnv, pipelineTargetEnvDefined := pipeline.ObjectMeta.Labels[kube.RadixEnvLabel]
	if !pipelineTargetEnvDefined {
		return nil, fmt.Errorf("missing target environment in labels of the pipeline %s", pipeline.Name)
	}

	log.Debugf("run pipelinerun for the tarfeg-environment %s", targetEnv)
	if _, ok := ctx.targetEnvironments[targetEnv]; !ok {
		return nil, fmt.Errorf("missing target environment %s for the pipeline %s", targetEnv, pipeline.Name)
	}

	pipelineRun := ctx.buildPipelineRun(pipeline, targetEnv, timestamp)

	return ctx.tektonClient.TektonV1beta1().PipelineRuns(namespace).Create(context.Background(), &pipelineRun, metav1.CreateOptions{})
}

func (ctx *pipelineContext) buildPipelineRun(pipeline *v1beta1.Pipeline, targetEnv, timestamp string) v1beta1.PipelineRun {
	originalPipelineName := pipeline.ObjectMeta.Annotations[defaults.PipelineNameAnnotation]
	pipelineRunName := fmt.Sprintf("radix-pipelinerun-%s-%s-%s", getShortName(targetEnv), timestamp, ctx.hash)
	pipelineParams := ctx.getPipelineParams(pipeline, targetEnv)
	pipelineRun := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipelineRunName,
			Labels: labels.GetLabelsForEnvironment(ctx, targetEnv),
			Annotations: map[string]string{
				kube.RadixBranchAnnotation:      ctx.env.GetBranch(),
				defaults.PipelineNameAnnotation: originalPipelineName,
			},
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{Name: pipeline.GetName()},
			Params:      pipelineParams,
		},
	}
	if ctx.ownerReference != nil {
		pipelineRun.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
	}
	return pipelineRun
}

func (ctx *pipelineContext) getPipelineParams(pipeline *v1beta1.Pipeline, targetEnv string) []v1beta1.Param {
	envVars := ctx.getEnvVars(targetEnv)
	pipelineParamsMap := getPipelineParamSpecsMap(pipeline)
	var pipelineParams []v1beta1.Param
	for envVarName, envVarValue := range envVars {
		paramSpec, envVarExistInParamSpecs := pipelineParamsMap[envVarName]
		if !envVarExistInParamSpecs {
			continue //Add to pipelineRun params only env-vars, existing in the pipeline paramSpecs
		}
		param := v1beta1.Param{
			Name:  envVarName,
			Value: v1beta1.ArrayOrString{Type: paramSpec.Type},
		}
		if param.Value.Type == v1beta1.ParamTypeArray {
			param.Value.ArrayVal = strings.Split(envVarValue, ",")
		} else {
			param.Value.StringVal = envVarValue
		}
		pipelineParams = append(pipelineParams, param)
	}
	return pipelineParams
}

func getPipelineParamSpecsMap(pipeline *v1beta1.Pipeline) map[string]v1beta1.ParamSpec {
	paramSpecMap := make(map[string]v1beta1.ParamSpec)
	for _, paramSpec := range pipeline.PipelineSpec().Params {
		paramSpecMap[paramSpec.Name] = paramSpec
	}
	return paramSpecMap
}

// WaitForCompletionOf Will wait for job to complete
func (ctx *pipelineContext) WaitForCompletionOf(pipelineRuns map[string]*v1beta1.PipelineRun) error {
	stop := make(chan struct{})
	defer close(stop)

	if len(pipelineRuns) == 0 {
		return nil
	}

	errChan := make(chan error)

	kubeInformerFactory := tektonInformerFactory.NewSharedInformerFactoryWithOptions(ctx.tektonClient, time.Second*5, tektonInformerFactory.WithNamespace(ctx.GetEnv().GetAppNamespace()))
	genericInformer, err := kubeInformerFactory.ForResource(v1beta1.SchemeGroupVersion.WithResource("pipelineruns"))
	if err != nil {
		return err
	}
	informer := genericInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, cur interface{}) {
			run, success := cur.(*v1beta1.PipelineRun)
			if !success {
				return
			}
			pipelineRun, ok := pipelineRuns[run.GetName()]
			if !ok {
				return
			}
			if pipelineRun.GetName() == run.GetName() && pipelineRun.GetNamespace() == run.GetNamespace() && run.Status.PipelineRunStatusFields.CompletionTime != nil {
				conditions := sortByTimestampDesc(run.Status.Conditions)
				if len(conditions) == 0 {
					return
				}
				delete(pipelineRuns, run.GetName())
				lastCondition := conditions[0]
				switch {
				case lastCondition.IsTrue():
					log.Infof("pipelineRun completed: %s", lastCondition.Message)
				default:
					log.Errorf("pipelineRun status reason %s. %s", lastCondition.Reason,
						lastCondition.Message)
				}
				if len(pipelineRuns) == 0 {
					errChan <- nil
				}
			} else {
				log.Debugf("Ongoing - PipelineRun has not completed yet")
			}
		},
		DeleteFunc: func(old interface{}) {
			run, success := old.(*v1beta1.PipelineRun)
			if !success {
				return
			}
			pipelineRun, ok := pipelineRuns[run.GetName()]
			if !ok {
				return
			}
			if pipelineRun.GetNamespace() == run.GetNamespace() {
				delete(pipelineRuns, run.GetName())
				errChan <- errors.New("PipelineRun failed - Job deleted")
			}
		},
	})
	go informer.Run(stop)
	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		errChan <- fmt.Errorf("timed out waiting for caches to sync")
	}

	err = <-errChan
	return err
}

func sortByTimestampDesc(conditions knative.Conditions) knative.Conditions {
	sort.Slice(conditions, func(i, j int) bool {
		return isC1BeforeC2(&conditions[j], &conditions[i])
	})
	return conditions
}

func isC1BeforeC2(c1 *knativeApis.Condition, c2 *knativeApis.Condition) bool {
	return c1.LastTransitionTime.Inner.Before(&c2.LastTransitionTime.Inner)
}
