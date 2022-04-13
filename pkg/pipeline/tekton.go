package pipeline

import (
	"context"
	"fmt"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (ctx pipelineContext) PrepareTektonPipelineJob() error {
	ra := ctx.radixApplication
	if ra == nil {
		return fmt.Errorf("no RadixApplication")
	}
	appName := ctx.radixApplication.GetName()
	namespace := utils.GetAppNamespace(appName)
	timestamp := time.Now().Format("20060102150405")
	imageTag := ctx.env.GetBranch()
	branch := ctx.env.GetBranch()
	radixPipelineRun := ctx.env.GetRadixPipelineRun()
	hash := strings.ToLower(utils.RandStringStrSeed(5, radixPipelineRun))
	task, err := ctx.createTask("task1", timestamp, imageTag, hash, namespace, radixPipelineRun)
	if err != nil {
		return err
	}
	log.Debugf("created task %s", task.Name)
	pipeline, err := ctx.createPipeline(namespace, timestamp, imageTag, hash, appName, branch, radixPipelineRun, task)
	if err != nil {
		return err
	}
	log.Debugf("created pipeline %s with %d tasks", pipeline.Name, len(pipeline.Spec.Tasks))

	return nil
}

func (ctx pipelineContext) createPipelineRun(namespace, timestamp, imageTag, hash, appName, branch string, pipeline *v1beta1.Pipeline) (*v1beta1.PipelineRun, error) {
	pipelineRunName := fmt.Sprintf("tekton-pipeline-run-%s-%s-%s", timestamp, imageTag, hash)
	pipelineRun := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipelineRunName,
			Labels: getLabels(appName, imageTag, hash, ""),
			Annotations: map[string]string{
				kube.RadixBranchAnnotation: branch,
			},
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{
				Name: pipeline.GetName(),
			},
		},
	}
	return ctx.tektonClient.TektonV1beta1().PipelineRuns(namespace).Create(context.Background(), &pipelineRun,
		metav1.CreateOptions{})
}

func (ctx pipelineContext) createPipeline(namespace, timestamp, imageTag, hash, appName, branch, radixPipelineRunLabel string, tasks ...*v1beta1.Task) (*v1beta1.Pipeline, error) {
	taskList := v1beta1.PipelineTaskList{}
	for _, task := range tasks {
		taskList = append(taskList, v1beta1.PipelineTask{
			Name: task.Name,
			TaskRef: &v1beta1.TaskRef{
				Name: task.Name,
			},
		},
		)
	}
	pipelineName := fmt.Sprintf("tekton-pipeline-%s-%s-%s", timestamp, imageTag, hash)
	pipeline := v1beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipelineName,
			Labels: getLabels(appName, imageTag, hash, radixPipelineRunLabel),
			Annotations: map[string]string{
				kube.RadixBranchAnnotation: branch,
			},
		},
		Spec: v1beta1.PipelineSpec{Tasks: taskList},
	}
	return ctx.tektonClient.TektonV1beta1().Pipelines(namespace).Create(context.Background(), &pipeline,
		metav1.CreateOptions{})
}

func (ctx pipelineContext) createTask(appName, timestamp, imageTag, hash, namespace, radixPipelineRun string) (*v1beta1.Task, error) {
	taskName := fmt.Sprintf("tekton-task-%s-%s-%s-%s", appName, timestamp, imageTag, hash)
	task := v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:   taskName,
			Labels: getLabels(appName, imageTag, hash, radixPipelineRun),
		},
		Spec: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{
				{
					Container: corev1.Container{
						Name:    "task1-container1",
						Image:   "bash:latest",
						Command: []string{"echo"},
						Args:    []string{"run task1"},
					},
				},
				//v1beta1.Step{
				//    Container: corev1.Container{
				//        Name:    "task1-container2",
				//        Image:   "bash:latest",
				//        Command: []string{"exit"},
				//        Args:    []string{"0"},
				//    },
				//},
			},
		},
	}
	return ctx.tektonClient.TektonV1beta1().Tasks(namespace).Create(context.Background(), &task,
		metav1.CreateOptions{})
}

func getLabels(appName, imageTag, hash, radixPipelineRun string) map[string]string {
	return map[string]string{
		kube.RadixBuildLabel:       fmt.Sprintf("%s-%s-%s", appName, imageTag, hash),
		kube.RadixAppLabel:         appName,
		kube.RadixImageTagLabel:    imageTag,
		kube.RadixJobTypeLabel:     kube.RadixJobTypeBuild,
		kube.RadixPipelineRunLabel: radixPipelineRun,
	}
}
