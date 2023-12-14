package wait

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/equinor/radix-tekton/pkg/models/env"
	log "github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonInformerFactory "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
	knativeApis "knative.dev/pkg/apis"
	knative "knative.dev/pkg/apis/duck/v1"
)

type PipelineRunsCompletionWaiter interface {
	Wait(pipelineRuns map[string]*pipelinev1.PipelineRun, env env.Env) error
}

func NewPipelineRunsCompletionWaiter(tektonClient tektonclient.Interface) PipelineRunsCompletionWaiter {
	return PipelineRunsCompletionWaiterFunc(func(pipelineRuns map[string]*pipelinev1.PipelineRun, env env.Env) error {
		return waitForCompletionOf(pipelineRuns, tektonClient, env)
	})
}

type PipelineRunsCompletionWaiterFunc func(pipelineRuns map[string]*pipelinev1.PipelineRun, env env.Env) error

func (f PipelineRunsCompletionWaiterFunc) Wait(pipelineRuns map[string]*pipelinev1.PipelineRun, env env.Env) error {
	return f(pipelineRuns, env)
}

// WaitForCompletionOf Will wait for job to complete
func waitForCompletionOf(pipelineRuns map[string]*pipelinev1.PipelineRun, tektonClient tektonclient.Interface, env env.Env) error {
	stop := make(chan struct{})
	defer close(stop)

	if len(pipelineRuns) == 0 {
		return nil
	}

	errChan := make(chan error)

	kubeInformerFactory := tektonInformerFactory.NewSharedInformerFactoryWithOptions(tektonClient, time.Second*5, tektonInformerFactory.WithNamespace(env.GetAppNamespace()))
	genericInformer, err := kubeInformerFactory.ForResource(pipelinev1.SchemeGroupVersion.WithResource("pipelineruns"))
	if err != nil {
		return err
	}
	informer := genericInformer.Informer()
	_, _ = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, cur interface{}) {
			run, success := cur.(*pipelinev1.PipelineRun)
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
				if strings.EqualFold(lastCondition.Reason, "Failed") {
					errChan <- errors.New("PipelineRun failed")
					return
				}
				switch {
				case lastCondition.Reason == pipelinev1.PipelineRunReasonCompleted.String():
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
			run, success := old.(*pipelinev1.PipelineRun)
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
		return isCondition1BeforeCondition2(&conditions[j], &conditions[i])
	})
	return conditions
}

func isCondition1BeforeCondition2(c1 *knativeApis.Condition, c2 *knativeApis.Condition) bool {
	return c1.LastTransitionTime.Inner.Before(&c2.LastTransitionTime.Inner)
}
