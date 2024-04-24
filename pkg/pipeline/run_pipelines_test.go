package pipeline_test

import (
	"context"
	"fmt"
	"testing"

	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-operator/pkg/apis/config/dnsalias"
	radixv1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/equinor/radix-tekton/pkg/defaults"
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
	"sigs.k8s.io/yaml"
)

const (
	appName              = "test-app"
	env1                 = "dev1"
	env2                 = "dev2"
	branchMain           = "main"
	radixImageTag        = "tag-123"
	radixPipelineJobName = "pipeline-job-123"
	radixConfigMapName   = "cm-name"
	someAzureClientId    = "some-azure-client-id"
)

var (
	RadixApplication = `apiVersion: radix.equinor.com/v1
kind: RadixApplication
metadata: 
  name: test-app
spec:
  environments:
  - name: dev
  - name: prod
`
)

func Test_RunPipeline_Has_ServiceAccount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	envMock := mockEnv(mockCtrl)
	kubeClient, rxClient, tknClient := test.Setup()
	completionWaiter := wait.NewMockPipelineRunsCompletionWaiter(mockCtrl)
	completionWaiter.EXPECT().Wait(gomock.Any(), gomock.Any()).AnyTimes()
	ctx := pipeline.NewPipelineContext(kubeClient, rxClient, tknClient, envMock, pipeline.WithPipelineRunsWaiter(completionWaiter))

	_, err := kubeClient.CoreV1().ConfigMaps(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: radixConfigMapName},
		Data: map[string]string{
			pipelineDefaults.PipelineConfigMapContent: RadixApplication,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = rxClient.RadixV1().RadixRegistrations().Create(context.TODO(), &radixv1.RadixRegistration{
		ObjectMeta: metav1.ObjectMeta{Name: appName}, Spec: radixv1.RadixRegistrationSpec{}}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = tknClient.TektonV1().Pipelines(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &pipelinev1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:   radixPipelineJobName,
			Labels: labels.GetLabelsForEnvironment(ctx, env1),
		},
		Spec: pipelinev1.PipelineSpec{},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	err = ctx.RunPipelinesJob()
	require.NoError(t, err)

	l, err := tknClient.TektonV1().PipelineRuns(ctx.GetEnv().GetAppNamespace()).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, l.Items)

	expected := utils.GetSubPipelineServiceAccountName(env1)
	actual := l.Items[0].Spec.TaskRunTemplate.ServiceAccountName
	assert.Equal(t, expected, actual)
}

func Test_RunPipeline_ApplyEnvVars(t *testing.T) {
	type scenario struct {
		name                          string
		pipelineSpec                  pipelinev1.PipelineSpec
		appEnvBuilder                 []utils.ApplicationEnvironmentBuilder
		buildVariables                radixv1.EnvVarsMap
		buildSubPipeline              utils.SubPipelineBuilder
		expectedPipelineRunParamNames map[string]string
	}

	scenarios := []scenario{
		{name: "no env vars and secrets",
			pipelineSpec:  pipelinev1.PipelineSpec{},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1)},
		},
		{name: "task uses common env vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
				},
			},
			appEnvBuilder:                 []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1)},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common", "var3": "value3common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1common", "var3": "value3common", "var4": "value4default"},
		},
		{name: "task do not use common env vars because of an empty sub-pipeline",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value1default"}},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
				},
			},
			appEnvBuilder:                 []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1)},
			buildSubPipeline:              utils.NewSubPipelineBuilder(),
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common", "var3": "value3common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1default", "var3": "value3default", "var4": "value4default"},
		},
		{name: "task uses common env vars from sub-pipeline, ignores build variables",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value1default"}},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			buildSubPipeline:              utils.NewSubPipelineBuilder().WithEnvVars(map[string]string{"var3": "value3sp", "var4": "value4sp"}),
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common", "var3": "value3common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1default", "var3": "value3sp", "var4": "value4sp", "var5": "value5default"},
		},
		{name: "task uses environment env vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1).
				WithEnvVars(map[string]string{"var3": "value3env", "var4": "value4env"})},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1common", "var3": "value3env", "var4": "value4env", "var5": "value5default"},
		},
		{name: "task uses environment env vars, not other environment env-vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{
				utils.NewApplicationEnvironmentBuilder().WithName(env1).
					WithEnvVars(map[string]string{"var3": "value3env", "var4": "value4env"}),
				utils.NewApplicationEnvironmentBuilder().WithName(env2).
					WithEnvVars(map[string]string{"var3": "value3env2", "var4": "value4env2"})},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1common", "var3": "value3env", "var4": "value4env", "var5": "value5default"},
		},
		{name: "task uses environment sub-pipeline env vars, ignores environment build variables",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1).
				WithSubPipeline(utils.NewSubPipelineBuilder().WithEnvVars(map[string]string{"var3": "value3sp-env", "var4": "value4sp-env"}))},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1common", "var3": "value3sp-env", "var4": "value4sp-env", "var5": "value5default"},
		},
		{name: "task do not use environment sub-pipeline env vars, because of empty env sub-pipeline",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value1default"}},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1).
				WithSubPipeline(utils.NewSubPipelineBuilder())},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common"},
			expectedPipelineRunParamNames: map[string]string{"var1": "value1common", "var3": "value3default", "var4": "value4default", "var5": "value5default"},
		},
		{name: "task uses environment sub-pipeline env vars, build sub-pipeline env vars, ignores environment build variables and build env vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1).
				WithSubPipeline(utils.NewSubPipelineBuilder().WithEnvVars(map[string]string{"var3": "value3sp-env", "var4": "value4sp-env"}))},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common", "var4": "value4common"},
			buildSubPipeline:              utils.NewSubPipelineBuilder().WithEnvVars(map[string]string{"var1": "value1sp", "var4": "value4sp"}),
			expectedPipelineRunParamNames: map[string]string{"var1": "value1sp", "var3": "value3sp-env", "var4": "value4sp-env", "var5": "value5default"},
		},
		{name: "task uses environment env vars, build sub-pipeline env vars, ignores environment build variables and build env vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: "var1", Type: pipelinev1.ParamTypeString},
					{Name: "var3", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value3default"}},
					{Name: "var4", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value4default"}},
					{Name: "var5", Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "value5default"}},
				},
			},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1).
				WithEnvVars(map[string]string{"var3": "value3env", "var4": "value4env"})},
			buildVariables:                map[string]string{"var1": "value1common", "var2": "value2common", "var4": "value4common"},
			buildSubPipeline:              utils.NewSubPipelineBuilder().WithEnvVars(map[string]string{"var1": "value1sp", "var4": "value4sp"}),
			expectedPipelineRunParamNames: map[string]string{"var1": "value1sp", "var3": "value3env", "var4": "value4env", "var5": "value5default"},
		},
	}

	for _, ts := range scenarios {
		t.Run(ts.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			envMock := mockEnv(mockCtrl)
			kubeClient, rxClient, tknClient := test.Setup()
			completionWaiter := wait.NewMockPipelineRunsCompletionWaiter(mockCtrl)
			completionWaiter.EXPECT().Wait(gomock.Any(), gomock.Any()).AnyTimes()
			ctx := pipeline.NewPipelineContext(kubeClient, rxClient, tknClient, envMock, pipeline.WithPipelineRunsWaiter(completionWaiter))

			raBuilder := utils.NewRadixApplicationBuilder().WithAppName(appName).
				WithBuildVariables(ts.buildVariables).
				WithSubPipeline(ts.buildSubPipeline).
				WithApplicationEnvironmentBuilders(ts.appEnvBuilder...)
			raContent, err := yaml.Marshal(raBuilder.BuildRA())
			require.NoError(t, err)
			fmt.Print(string(raContent))
			_, err = rxClient.RadixV1().RadixRegistrations().Create(context.TODO(), &radixv1.RadixRegistration{
				ObjectMeta: metav1.ObjectMeta{Name: appName}, Spec: radixv1.RadixRegistrationSpec{}}, metav1.CreateOptions{})
			require.NoError(t, err)
			_, err = kubeClient.CoreV1().ConfigMaps(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: radixConfigMapName},
				Data: map[string]string{
					"content": string(raContent),
				},
			}, metav1.CreateOptions{})

			_, err = tknClient.TektonV1().Pipelines(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &pipelinev1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{Name: radixPipelineJobName, Labels: labels.GetLabelsForEnvironment(ctx, env1)},
				Spec:       ts.pipelineSpec}, metav1.CreateOptions{})
			require.NoError(t, err)

			err = ctx.RunPipelinesJob()
			require.NoError(t, err)

			pipelineRunList, err := tknClient.TektonV1().PipelineRuns(ctx.GetEnv().GetAppNamespace()).List(context.TODO(), metav1.ListOptions{})
			require.NoError(t, err)
			assert.Len(t, pipelineRunList.Items, 1)
			pr := pipelineRunList.Items[0]
			assert.Len(t, pr.Spec.Params, len(ts.expectedPipelineRunParamNames), "mismatching pipelineRun.Spec.Params element count")
			for _, param := range pr.Spec.Params {
				expectedValue, ok := ts.expectedPipelineRunParamNames[param.Name]
				assert.True(t, ok, "unexpected param %s", param.Name)
				assert.Equal(t, expectedValue, param.Value.StringVal, "mismatching value in the param %s", param.Name)
				assert.Equal(t, pipelinev1.ParamTypeString, param.Value.Type, "mismatching type of the param %s", param.Name)
			}
		})
	}
}

func Test_RunPipeline_ApplyIdentity(t *testing.T) {
	type scenario struct {
		name                          string
		pipelineSpec                  pipelinev1.PipelineSpec
		appEnvBuilder                 []utils.ApplicationEnvironmentBuilder
		buildIdentity                 *radixv1.Identity
		buildSubPipeline              utils.SubPipelineBuilder
		expectedPipelineRunParamNames map[string]string
	}

	scenarios := []scenario{
		{name: "no env vars and secrets",
			pipelineSpec:  pipelinev1.PipelineSpec{},
			appEnvBuilder: []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1)},
		},
		{name: "task uses common env vars",
			pipelineSpec: pipelinev1.PipelineSpec{
				Params: []pipelinev1.ParamSpec{
					{Name: defaults.AzureClientIdEnvironmentVariable, Type: pipelinev1.ParamTypeString, Default: &pipelinev1.ParamValue{StringVal: "not-set"}},
				},
			},
			appEnvBuilder:                 []utils.ApplicationEnvironmentBuilder{utils.NewApplicationEnvironmentBuilder().WithName(env1)},
			buildIdentity:                 &radixv1.Identity{Azure: &radixv1.AzureIdentity{ClientId: someAzureClientId}},
			expectedPipelineRunParamNames: map[string]string{defaults.AzureClientIdEnvironmentVariable: someAzureClientId},
		},
	}

	for _, ts := range scenarios {
		t.Run(ts.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			envMock := mockEnv(mockCtrl)
			kubeClient, rxClient, tknClient := test.Setup()
			completionWaiter := wait.NewMockPipelineRunsCompletionWaiter(mockCtrl)
			completionWaiter.EXPECT().Wait(gomock.Any(), gomock.Any()).AnyTimes()
			ctx := pipeline.NewPipelineContext(kubeClient, rxClient, tknClient, envMock, pipeline.WithPipelineRunsWaiter(completionWaiter))

			raBuilder := utils.NewRadixApplicationBuilder().WithAppName(appName).
				WithApplicationEnvironmentBuilders(ts.appEnvBuilder...)
			if ts.buildIdentity != nil {
				raBuilder = raBuilder.WithSubPipeline(utils.NewSubPipelineBuilder().WithIdentity(ts.buildIdentity))
			}
			raContent, err := yaml.Marshal(raBuilder.BuildRA())
			require.NoError(t, err)
			_, err = rxClient.RadixV1().RadixRegistrations().Create(context.TODO(), &radixv1.RadixRegistration{
				ObjectMeta: metav1.ObjectMeta{Name: appName}, Spec: radixv1.RadixRegistrationSpec{}}, metav1.CreateOptions{})
			require.NoError(t, err)
			_, err = kubeClient.CoreV1().ConfigMaps(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: radixConfigMapName},
				Data: map[string]string{
					"content": string(raContent),
				},
			}, metav1.CreateOptions{})

			_, err = tknClient.TektonV1().Pipelines(ctx.GetEnv().GetAppNamespace()).Create(context.TODO(), &pipelinev1.Pipeline{
				ObjectMeta: metav1.ObjectMeta{Name: radixPipelineJobName, Labels: labels.GetLabelsForEnvironment(ctx, env1)},
				Spec:       ts.pipelineSpec}, metav1.CreateOptions{})
			require.NoError(t, err)

			err = ctx.RunPipelinesJob()
			require.NoError(t, err)

			pipelineRunList, err := tknClient.TektonV1().PipelineRuns(ctx.GetEnv().GetAppNamespace()).List(context.TODO(), metav1.ListOptions{})
			require.NoError(t, err)
			assert.Len(t, pipelineRunList.Items, 1)
			pr := pipelineRunList.Items[0]
			assert.Len(t, pr.Spec.Params, len(ts.expectedPipelineRunParamNames), "mismatching pipelineRun.Spec.Params element count")
			for _, param := range pr.Spec.Params {
				expectedValue, ok := ts.expectedPipelineRunParamNames[param.Name]
				assert.True(t, ok, "unexpected param %s", param.Name)
				assert.Equal(t, expectedValue, param.Value.StringVal, "mismatching value in the param %s", param.Name)
				assert.Equal(t, pipelinev1.ParamTypeString, param.Value.Type, "mismatching type of the param %s", param.Name)
			}
		})
	}
}

func mockEnv(mockCtrl *gomock.Controller) *env.MockEnv {
	mockEnv := env.NewMockEnv(mockCtrl)
	mockEnv.EXPECT().GetAppName().Return(appName).AnyTimes()
	mockEnv.EXPECT().GetRadixImageTag().Return(radixImageTag).AnyTimes()
	mockEnv.EXPECT().GetRadixPipelineJobName().Return(radixPipelineJobName).AnyTimes()
	mockEnv.EXPECT().GetBranch().Return(branchMain).AnyTimes()
	mockEnv.EXPECT().GetAppNamespace().Return(utils.GetAppNamespace(appName)).AnyTimes()
	mockEnv.EXPECT().GetRadixPipelineType().Return(radixv1.Deploy).AnyTimes()
	mockEnv.EXPECT().GetRadixConfigMapName().Return(radixConfigMapName).AnyTimes()
	mockEnv.EXPECT().GetRadixDeployToEnvironment().Return(env1).AnyTimes()
	mockEnv.EXPECT().GetDNSConfig().Return(&dnsalias.DNSConfig{}).AnyTimes()
	mockEnv.EXPECT().GetRadixConfigBranch().Return(env1).AnyTimes()
	return mockEnv
}
