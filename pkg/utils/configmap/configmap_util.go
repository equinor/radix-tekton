package configmap

import (
	"context"
	"fmt"
	"os"

	"github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/models/config"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateFromRadixConfigFile Creates a configmap by name from file and returns as content
func CreateFromRadixConfigFile(cfg config.Config) (string, error) {
	content, err := os.ReadFile(cfg.GetRadixConfigFileName())
	if err != nil {
		return "", fmt.Errorf("could not find or read config yaml file \"%s\"", cfg.GetRadixConfigFileName())
	}
	return string(content), nil
}

// CreateGitConfigFromGitRepository create configmap with git repository information
func CreateGitConfigFromGitRepository(cfg config.Config, kubeClient kubernetes.Interface, targetCommitHash, gitTags string) error {
	_, err := kubeClient.CoreV1().ConfigMaps(cfg.GetAppNamespace()).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.GetGitConfigMapName(),
				Namespace: cfg.GetAppNamespace(),
				Labels:    map[string]string{kube.RadixJobNameLabel: cfg.GetRadixPipelineJobName()},
			},
			Data: map[string]string{
				defaults.RadixGitCommitHashKey: targetCommitHash,
				defaults.RadixGitTagsKey:       gitTags,
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Debugf("Created ConfigMap %s", cfg.GetGitConfigMapName())
	return nil
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
