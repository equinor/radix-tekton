package commithash

import (
	"context"

	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type provider struct {
	radixClient  radixclient.Interface
	appName      string
	environments []string
}

// EnvCommitHashMap Last commit hashes in environments
type EnvCommitHashMap map[string]string

type Provider interface {
	// GetLastCommitHashesForEnvironments  Gets last successful environment Radix deployment commit hashes
	GetLastCommitHashesForEnvironments() (EnvCommitHashMap, error)
}

// NewProvider New instance of the Radix deployment commit hashes provider
func NewProvider(radixClient radixclient.Interface, appName string, environments []string) Provider {
	return &provider{
		radixClient:  radixClient,
		appName:      appName,
		environments: environments,
	}
}

func (provider *provider) GetLastCommitHashesForEnvironments() (EnvCommitHashMap, error) {
	envCommitMap := make(EnvCommitHashMap)
	appNamespace := utils.GetAppNamespace(provider.appName)
	jobTypeMap, err := getJobTypeMap(provider.radixClient, appNamespace)
	if err != nil {
		return nil, err
	}
	for _, envName := range provider.environments {
		radixDeployments, err := provider.getRadixDeploymentsForEnvironment(envName)
		if err != nil {
			return nil, err
		}
		envCommitMap[envName] = getLastRadixDeploymentCommitHash(radixDeployments, jobTypeMap)
	}
	return envCommitMap, nil
}

func (provider *provider) getRadixDeploymentsForEnvironment(name string) ([]v1.RadixDeployment, error) {
	namespace := utils.GetEnvironmentNamespace(provider.appName, name)
	deployments := provider.radixClient.RadixV1().RadixDeployments(namespace)
	radixDeploymentList, err := deployments.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return radixDeploymentList.Items, nil
}

func getLastRadixDeploymentCommitHash(radixDeployments []v1.RadixDeployment, jobTypeMap map[string]v1.RadixPipelineType) string {
	var lastRadixDeployment *v1.RadixDeployment
	for _, radixDeployment := range radixDeployments {
		pipeLineType, ok := jobTypeMap[radixDeployment.GetLabels()[kube.RadixJobNameLabel]]
		if !ok || pipeLineType != v1.BuildDeploy {
			continue
		}
		if lastRadixDeployment == nil || timeIsBefore(lastRadixDeployment.Status.ActiveFrom, radixDeployment.Status.ActiveFrom) {
			lastRadixDeployment = &radixDeployment
		}
	}
	if lastRadixDeployment == nil {
		return ""
	}
	if commitHash, ok := lastRadixDeployment.GetAnnotations()[kube.RadixCommitAnnotation]; ok && len(commitHash) > 0 {
		return commitHash
	}
	return lastRadixDeployment.GetLabels()[kube.RadixCommitLabel]
}

func getJobTypeMap(radixClient radixclient.Interface, appNamespace string) (map[string]v1.RadixPipelineType, error) {
	radixJobList, err := radixClient.RadixV1().RadixJobs(appNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	jobMap := make(map[string]v1.RadixPipelineType)
	for _, rj := range radixJobList.Items {
		jobMap[rj.GetName()] = rj.Spec.PipeLineType
	}
	return jobMap, nil
}

func timeIsBefore(time1 metav1.Time, time2 metav1.Time) bool {
	if time1.IsZero() {
		return true
	}
	if time2.IsZero() {
		return false
	}
	return time1.Before(&time2)
}
