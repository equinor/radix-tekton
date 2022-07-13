package configmap

import (
	"context"
	"fmt"
	"github.com/equinor/radix-tekton/pkg/utils/git"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"

	"github.com/equinor/radix-tekton/pkg/models/env"
	corev1 "k8s.io/api/core/v1"
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
				"content": configFileContent,
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
	gitCommitHash, gitTags, err := git.GetGitCommitHashAndTags("/workspace/.git")

	_, err = kubeClient.CoreV1().ConfigMaps(env.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.GetGitConfigMapName(),
				Namespace: env.GetAppNamespace(),
			},
			Data: map[string]string{
				"tekton-pipeline": "true",
				"git-commit-hash": gitCommitHash,
				"git-commit-tags": fmt.Sprintf("(%s)", gitTags),
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Created ConfigMap %s", env.GetRadixConfigMapName())
	return nil
}

//GetRadixConfigFromConfigMap Get Radix config from the ConfigMap
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
