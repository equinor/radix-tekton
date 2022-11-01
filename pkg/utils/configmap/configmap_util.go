package configmap

import (
	"context"
	"fmt"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/equinor/radix-operator/pkg/apis/defaults"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/utils/git"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type lastEnvironmentDeployCommit struct {
	envName    string
	commitHash string
}

// CreateFromRadixConfigFile Creates a configmap by name from file and returns as content
func CreateFromRadixConfigFile(kubeClient kubernetes.Interface, env env.Env) (string, error) {
	content, err := readConfigFile(env.GetRadixConfigFileName())
	if err != nil {
		return "", fmt.Errorf("could not find or read config yaml file \"%s\"", env.GetRadixConfigFileName())
	}

	configFileContent := string(content)
	_, err = kubeClient.CoreV1().ConfigMaps(env.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.GetRadixConfigMapName(),
				Namespace: env.GetAppNamespace(),
			},
			Data: map[string]string{
				"tekton-pipeline": "true",
				"content":         configFileContent,
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return "", err
	}
	log.Debugf("Created ConfigMap %s", env.GetRadixConfigMapName())
	return configFileContent, nil
}

// CreateGitConfigFromGitRepository create configmap with git repository information
func CreateGitConfigFromGitRepository(kubeClient kubernetes.Interface, radixClient radixclient.Interface, appName string, env env.Env, environments []string) error {
	workspace := env.GetGitRepositoryWorkspace()
	targetCommitHash, err := getGitCommitHash(workspace, env)
	if err != nil {
		return err
	}

	gitDir := workspace + "/.git"
	gitTags, err := git.GetGitCommitTags(gitDir, targetCommitHash)
	if err != nil {
		return err
	}

	beforeCommitHash, err := getLastSuccessfulEnvironmentDeployCommits(radixClient, appName, environments)
	if err != nil {
		return err
	}
	changedFolders, radixConfigChanged, err := git.GetGitAffectedResourcesBetweenCommits(gitDir, targetCommitHash, beforeCommitHash, env.GetRadixConfigFileName(), env.GetRadixConfigBranch())
	if err != nil {
		return err
	}
	_, err = kubeClient.CoreV1().ConfigMaps(env.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.GetGitConfigMapName(),
				Namespace: env.GetAppNamespace(),
			},
			Data: map[string]string{
				defaults.RadixGitCommitHashKey:             targetCommitHash,
				defaults.RadixGitTagsKey:                   gitTags,
				defaults.RadixGitChangedFolders:            getFolderListAsString(changedFolders),
				defaults.RadixGitChangedChangedRadixConfig: strconv.FormatBool(radixConfigChanged),
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Created ConfigMap %s", env.GetGitConfigMapName())
	return nil
}

func getLastSuccessfulEnvironmentDeployCommits(radixClient radixclient.Interface, appName string, environments []string) (map[string]string, error) {
	envCommitMap := make(map[string]string)
	appNamespace := utils.GetAppNamespace(appName)
	radixJobList, err := radixClient.RadixV1().RadixJobs(appNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	jobMap := make(map[string]v1.RadixPipelineType)
	for _, rj := range radixJobList.Items {
		jobMap[rj.GetName()] = rj.Spec.PipeLineType
	}
	for _, envName := range environments {
		namespace := utils.GetEnvironmentNamespace(appName, envName)
		deployments := radixClient.RadixV1().RadixDeployments(namespace)
		radixDeploymentList, err := deployments.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var lastRadixDeploy *v1.RadixDeployment
		for _, rd := range radixDeploymentList.Items {
			if pipeLineType, ok := jobMap[rd.GetLabels()[kube.RadixJobNameLabel]]; ok {
				if lastRadixDeploy == nil || lastRadixDeploy.Status.ActiveFrom.Time < rd.Status.ActiveFrom.Time {
					lastRadixDeploy = &rd
				}
			}
		}
	}
	return envCommitMap, nil
}

func getFolderListAsString(changedFolders []string) string {
	for i, name := range changedFolders {
		changedFolders[i] = strings.ReplaceAll(name, ",", "\\,")
	}
	return strings.Join(changedFolders, ",")
}

func getGitCommitHash(workspace string, e env.Env) (string, error) {
	webhookCommitId := e.GetWebhookCommitId()
	if webhookCommitId != "" {
		log.Debugf("got git commit hash %s from env var %s", webhookCommitId, defaults.RadixGithubWebhookCommitId)
		return webhookCommitId, nil
	}
	branchName := e.GetBranch()
	log.Debugf("determining git commit hash of HEAD of branch %s", branchName)
	gitCommitHash, err := git.GetGitCommitHashFromHead(workspace+"/.git", branchName)
	log.Debugf("got git commit hash %s from HEAD of branch %s", gitCommitHash, branchName)
	return gitCommitHash, err
}

// GetRadixConfigFromConfigMap Get Radix config from the ConfigMap
func GetRadixConfigFromConfigMap(kubeClient kubernetes.Interface, namespace, configMapName string) (string, error) {
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("no ConfigMap %s", configMapName)
		}
		return "", err
	}
	if configMap.Data == nil {
		return "", getNoRadixConfigInConfigMap(configMapName)
	}
	content, ok := configMap.Data["content"]
	if !ok {
		return "", getNoRadixConfigInConfigMap(configMapName)
	}
	return content, nil
}

func getNoRadixConfigInConfigMap(configMapName string) error {
	return fmt.Errorf("no RadixConfig in the ConfigMap %s", configMapName)
}

func readConfigFile(filename string) ([]byte, error) {
	var content []byte
	var err error
	for _, filename := range filenameCandidates(filename) {
		content, err = ioutil.ReadFile(filename)
		if err == nil {
			break
		}
	}
	return content, err
}

func filenameCandidates(filename string) []string {
	if strings.HasSuffix(filename, ".yaml") {
		filename = filename[:len(filename)-5]
	} else if strings.HasSuffix(filename, ".yml") {
		filename = filename[:len(filename)-4]
	}

	return []string{
		filename + ".yaml",
		filename + ".yml",
		filename,
	}
}
