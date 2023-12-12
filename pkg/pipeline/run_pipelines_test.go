package pipeline_test

import (
	"context"
	"testing"

	radixv1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/equinor/radix-tekton/pkg/internal/wait"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	"github.com/equinor/radix-tekton/pkg/utils/labels"
	"github.com/equinor/radix-tekton/pkg/utils/test"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// assert.Equal(t, utils.GetSubPipelineServiceAccountName(envDev), pipeline.)

const (
	appName              = "test-app"
	branchMain           = "main"
	radixImageTag        = "tag-123"
	radixPipelineJobName = "pipeline-job-123"
	radixConfigMapName   = "cm-name"
)

var (
	RadixApplication = `apiVersion: radix.equinor.com/v1
kind: RadixApplication
metadata: 
  name: radix-application
spec:
  environments:
  - name: dev
  - name: prod
`
)

func Test_RunPipeline_Has_ServiceAccount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockEnv := env.NewMockEnv(mockCtrl)
	mockEnv.EXPECT().GetAppName().Return(appName).AnyTimes()
	mockEnv.EXPECT().GetRadixImageTag().Return(radixImageTag).AnyTimes()
	mockEnv.EXPECT().GetRadixPipelineJobName().Return(radixPipelineJobName).AnyTimes()
	mockEnv.EXPECT().GetBranch().Return(branchMain).AnyTimes()
	mockEnv.EXPECT().GetAppNamespace().Return(utils.GetAppNamespace(appName)).AnyTimes()
	mockEnv.EXPECT().GetRadixPipelineType().Return(radixv1.Deploy).AnyTimes()
	mockEnv.EXPECT().GetRadixConfigMapName().Return(radixConfigMapName).AnyTimes()
	mockEnv.EXPECT().GetRadixDeployToEnvironment().Return("dev").AnyTimes()
	kubeclient, rxclient, tknclient := test.Setup()
	ctx := pipeline.NewPipelineContext(kubeclient, rxclient, tknclient, mockEnv)
	waiter := wait.NewMockPipelineRunsCompletionWaiter(mockCtrl)
	waiter.EXPECT().Wait(gomock.Any(), gomock.Any()).AnyTimes()
	ctx.WithPipelineRunsWaiter(waiter)

	_, err := kubeclient.CoreV1().ConfigMaps(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: radixConfigMapName},
		Data: map[string]string{
			"content": RadixApplication,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = rxclient.RadixV1().RadixRegistrations().Create(context.TODO(), &radixv1.RadixRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "radix-application",
		},
		Spec: radixv1.RadixRegistrationSpec{},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = tknclient.TektonV1().Pipelines(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &pipelinev1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:   radixPipelineJobName,
			Labels: labels.GetLabelsForEnvironment(ctx, "dev"),
		},
		Spec: pipelinev1.PipelineSpec{},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	err = ctx.RunPipelinesJob()
	require.NoError(t, err)

	l, err := tknclient.TektonV1().PipelineRuns(ctx.GetEnv().GetAppNamespace()).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, l.Items)

	expected := utils.GetSubPipelineServiceAccountName("dev")
	actual := l.Items[0].Spec.TaskRunTemplate.ServiceAccountName
	assert.Equal(t, expected, actual)
}
