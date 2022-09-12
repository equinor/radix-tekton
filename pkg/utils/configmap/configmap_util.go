package configmap

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/equinor/radix-operator/pkg/apis/defaults"
	operatorGit "github.com/equinor/radix-operator/pkg/apis/utils/git"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/utils/git"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

func CreateFromGitRepository(kubeClient kubernetes.Interface, env env.Env) error {
	gitCommitHash, err := getGitCommitHash(env)
	if err != nil {
		return err
	}

	gitTags, err := git.GetGitCommitTags("/Users/SSMOL/dev/equinor/radix-mini-app-with-tekton/.git", gitCommitHash)
	//gitTags, err := git.GetGitCommitTags(operatorGit.Workspace+"/.git", gitCommitHash)
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
				defaults.RadixGitCommitHashKey: gitCommitHash,
				defaults.RadixGitTagsKey:       gitTags,
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Created ConfigMap %s", env.GetGitConfigMapName())
	return nil
}

func getGitCommitHash(e env.Env) (string, error) {
	webhookCommitId := e.GetWebhookCommitId()
	if webhookCommitId != "" {
		log.Debugf("got git commit hash %s from env var %s", webhookCommitId, defaults.RadixGithubWebhookCommitId)
		return webhookCommitId, nil
	}
	branchName := e.GetBranch()
	log.Debugf("determining git commit hash of HEAD of branch %s", branchName)
	gitCommitHash, err := git.GetGitCommitHashFromHead(operatorGit.Workspace+"/.git", branchName)
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
