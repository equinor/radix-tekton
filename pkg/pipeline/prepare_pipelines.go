package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/equinor/radix-operator/pkg/apis/kube"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctx pipelineContext) prepareTektonPipelineJob() error {
	namespace := ctx.env.GetAppNamespace()
	timestamp := time.Now().Format("20060102150405")
	task, err := ctx.createTask(timestamp, namespace)
	if err != nil {
		return err
	}
	log.Debugf("created task %s", task.Name)
	pipeline, err := ctx.createPipeline(namespace, timestamp, task)
	if err != nil {
		return err
	}
	log.Debugf("created pipeline %s with %d tasks", pipeline.Name, len(pipeline.Spec.Tasks))

	return nil
}

func (ctx pipelineContext) createPipeline(namespace, timestamp string, tasks ...*v1beta1.Task) (*v1beta1.Pipeline, error) {
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
	pipelineName := fmt.Sprintf("tekton-pipeline-%s-%s-%s", timestamp, ctx.env.GetRadixImageTag(), ctx.hash)
	pipeline := v1beta1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipelineName,
			Labels: ctx.getLabels(),
			Annotations: map[string]string{
				kube.RadixBranchAnnotation: ctx.env.GetBranch(),
			},
		},
		Spec: v1beta1.PipelineSpec{Tasks: taskList},
	}
	return ctx.tektonClient.TektonV1beta1().Pipelines(namespace).Create(context.Background(), &pipeline,
		metav1.CreateOptions{})
}

func (ctx pipelineContext) createTask(timestamp, namespace string) (*v1beta1.Task, error) {
	taskName := fmt.Sprintf("tekton-task-%s-%s-%s-%s", ctx.env.GetAppName(), timestamp, ctx.env.GetRadixImageTag(), ctx.hash)
	task := v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:   taskName,
			Labels: ctx.getLabels(),
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
				{
					Container: corev1.Container{
						Name:    "task1-container3",
						Image:   "bash:latest",
						Command: []string{"sleep"},
						Args:    []string{"5"},
					},
				},
			},
		},
	}
	return ctx.tektonClient.TektonV1beta1().Tasks(namespace).Create(context.Background(), &task,
		metav1.CreateOptions{})
}
