package configmap

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateFromFile Creates a configmap by name from file and returns as content
func CreateFromFile(kubeClient kubernetes.Interface, namespace, name, file string) (string, error) {
	content, err := readConfigFile(file)
	if err != nil {
		return "", fmt.Errorf("could not find or read config yaml file \"%s\"", file)
	}

	configFileContent := string(content)
	_, err = kubeClient.CoreV1().ConfigMaps(namespace).Create(
		context.Background(),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string]string{
				"content": configFileContent,
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		return "", err
	}

	return configFileContent, nil
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
