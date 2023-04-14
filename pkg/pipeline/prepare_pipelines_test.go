package pipeline

import (
	"context"
	commonUtils "github.com/equinor/radix-common/utils"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclientfake "github.com/equinor/radix-operator/pkg/client/clientset/versioned/fake"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/stretchr/testify/assert"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclientfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientfake "k8s.io/client-go/kubernetes/fake"
)

func Test_ComponentHasChangedSource(t *testing.T) {
	var testScenarios = []struct {
		description    string
		changedFolders []string
		sourceFolder   string
		expectedResult bool
	}{
		{
			description:    "sourceFolder is dot",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   ".",
			expectedResult: true,
		},
		{
			description:    "several dots and slashes",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "././",
			expectedResult: true,
		},
		{
			description:    "no changes in the sourceFolder folder with trailing slash",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "nonexistingdir/",
			expectedResult: false,
		},
		{
			description:    "no changes in the sourceFolder folder without slashes",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "nonexistingdir",
			expectedResult: false,
		},
		{
			description:    "real source dir with trailing slash",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src/",
			expectedResult: true,
		},
		{
			description:    "sourceFolder has surrounding slashes and leading dot",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "./src/",
			expectedResult: true,
		},
		{
			description:    "real source dir without trailing slash",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "./src",
			expectedResult: true,
		},
		{
			description:    "changes in the sourceFolder folder subfolder",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src",
			expectedResult: true,
		},
		{
			description:    "changes in the sourceFolder multiple element path subfolder",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src/subdir",
			expectedResult: true,
		},
		{
			description:    "changes in the sourceFolder multiple element path subfolder with trailing slash",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src/subdir/",
			expectedResult: true,
		},
		{
			description:    "no changes in the sourceFolder multiple element path subfolder",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src/subdir/water",
			expectedResult: false,
		},
		{
			description:    "changes in the sourceFolder multiple element path subfolder with trailing slash",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "src/subdir/water/",
			expectedResult: false,
		},
		{
			description:    "sourceFolder has name of changed folder",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "notebooks",
			expectedResult: true,
		},
		{
			description:    "empty sourceFolder is affected by any changes",
			changedFolders: []string{"src/some/subdir", "src/subdir/business_logic", "notebooks", "tests"},
			sourceFolder:   "",
			expectedResult: true,
		},
		{
			description:    "empty sourceFolder is affected by any changes",
			changedFolders: []string{"src/some/subdir"},
			sourceFolder:   "",
			expectedResult: true,
		},
		{
			description:    "sourceFolder sub-folder in the root",
			changedFolders: []string{".", "app1"},
			sourceFolder:   "./app1",
			expectedResult: true,
		},
		{
			description:    "sourceFolder sub-folder in the root, no changed folders",
			changedFolders: []string{},
			sourceFolder:   ".",
			expectedResult: false,
		},
		{
			description:    "sourceFolder sub-folder in empty, no changed folders",
			changedFolders: []string{},
			sourceFolder:   "",
			expectedResult: false,
		},
		{
			description:    "sourceFolder sub-folder in slash, no changed folders",
			changedFolders: []string{},
			sourceFolder:   "/",
			expectedResult: false,
		},
		{
			description:    "sourceFolder sub-folder in slash with dot, no changed folders",
			changedFolders: []string{},
			sourceFolder:   "/.",
			expectedResult: false,
		},
	}

	var applicationComponent v1.RadixComponent

	for _, testScenario := range testScenarios {
		t.Run(testScenario.description, func(t *testing.T) {
			applicationComponent =
				utils.AnApplicationComponent().
					WithName("client-component-1").
					WithSourceFolder(testScenario.sourceFolder).
					BuildComponent()
			sourceHasChanged := componentHasChangedSource("someEnv", &applicationComponent, testScenario.changedFolders)
			assert.Equal(t, testScenario.expectedResult, sourceHasChanged)
		})
	}
}

func Test_pipelineContext_createPipeline(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	type fields struct {
		env                env.Env
		radixApplication   *v1.RadixApplication
		targetEnvironments map[string]bool
		hash               string
		ownerReference     *metav1.OwnerReference
	}
	type args struct {
		envName   string
		pipeline  *tekton.Pipeline
		tasks     []tekton.Task
		timestamp string
	}
	const (
		appName              = "test-app"
		envDev               = "dev"
		branchMain           = "main"
		radixImageTag        = "tag-123"
		radixPipelineJobName = "pipeline-job-123"
		hash                 = "some-hash"
	)

	scenarios := []struct {
		name           string
		fields         fields
		args           args
		wantErr        func(t *testing.T, err error)
		assertScenario func(t *testing.T, ctx *pipelineContext)
	}{
		{
			name: "one default task",
			fields: fields{
				targetEnvironments: map[string]bool{envDev: true},
				hash:               hash,
				ownerReference:     &metav1.OwnerReference{Kind: "RadixApplication", Name: appName},
			},
			args: args{envName: envDev, pipeline: getTestPipeline(func(pipeline *tekton.Pipeline) { pipeline.ObjectMeta.Name = "pipeline1" }),
				tasks: []tekton.Task{*getTestTask(func(task *tekton.Task) {
					task.ObjectMeta.Name = "task1"
					task.Spec = tekton.TaskSpec{
						Steps:        []tekton.Step{{Name: "step1"}},
						Sidecars:     []tekton.Sidecar{{Name: "sidecar1"}},
						StepTemplate: &tekton.StepTemplate{Image: "image1"},
					}
				})}, timestamp: "2020-01-01T00:00:00Z"},
			wantErr: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
			assertScenario: func(t *testing.T, ctx *pipelineContext) {
				pipeline, err := ctx.tektonClient.TektonV1().Pipelines(ctx.env.GetAppNamespace()).Get(context.Background(), "pipeline1", metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, 1, len(pipeline.Spec.Tasks))
				task := pipeline.Spec.Tasks[0]
				assert.Equal(t, "task1", task.Name)
				assert.Equal(t, 1, len(task.TaskSpec.Steps))
				assert.Equal(t, "step1", task.TaskSpec.Steps[0].Name)
				assert.Equal(t, 1, len(task.TaskSpec.Sidecars))
				assert.Equal(t, "sidecar1", task.TaskSpec.Sidecars[0].Name)
				assert.Equal(t, "image1", task.TaskSpec.StepTemplate.Image)
				assert.NotNilf(t, pipeline.ObjectMeta.OwnerReferences, "Expected owner reference to be set")
				assert.Len(t, pipeline.ObjectMeta.OwnerReferences, 1)
				assert.Equal(t, "RadixApplication", pipeline.ObjectMeta.OwnerReferences[0].Kind)
				assert.Equal(t, appName, pipeline.ObjectMeta.OwnerReferences[0].Name)
			},
		},
		{
			name: "set SecurityContexts in task step",
			fields: fields{
				targetEnvironments: map[string]bool{envDev: true},
			},
			args: args{envName: envDev, pipeline: getTestPipeline(func(pipeline *tekton.Pipeline) { pipeline.ObjectMeta.Name = "pipeline1" }),
				tasks: []tekton.Task{*getTestTask(func(task *tekton.Task) {
					task.Spec = tekton.TaskSpec{
						Steps: []tekton.Step{{Name: "step1"}},
					}
				})}, timestamp: "2020-01-01T00:00:00Z"},
			wantErr: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
			assertScenario: func(t *testing.T, ctx *pipelineContext) {
				pipeline, err := ctx.tektonClient.TektonV1().Pipelines(ctx.env.GetAppNamespace()).Get(context.Background(), "pipeline1", metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, 1, len(pipeline.Spec.Tasks))
				task := pipeline.Spec.Tasks[0]
				assert.Equal(t, 1, len(task.TaskSpec.Steps))
				step := task.TaskSpec.Steps[0]
				assert.NotNil(t, step.SecurityContext)
				assert.Equal(t, commonUtils.BoolPtr(true), step.SecurityContext.RunAsNonRoot)
				assert.Equal(t, commonUtils.BoolPtr(false), step.SecurityContext.Privileged)
				assert.Equal(t, commonUtils.BoolPtr(false), step.SecurityContext.AllowPrivilegeEscalation)
				assert.NotNil(t, step.SecurityContext.Capabilities)
				assert.Equal(t, []corev1.Capability{"ALL"}, step.SecurityContext.Capabilities.Drop)
			},
		},
		{
			name: "set SecurityContexts in task sidecar",
			fields: fields{
				targetEnvironments: map[string]bool{envDev: true},
			},
			args: args{envName: envDev, pipeline: getTestPipeline(func(pipeline *tekton.Pipeline) { pipeline.ObjectMeta.Name = "pipeline1" }),
				tasks: []tekton.Task{*getTestTask(func(task *tekton.Task) {
					task.Spec = tekton.TaskSpec{
						Steps: []tekton.Step{{Name: "step1"}},
					}
				})}, timestamp: "2020-01-01T00:00:00Z"},
			wantErr: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
			assertScenario: func(t *testing.T, ctx *pipelineContext) {
				pipeline, err := ctx.tektonClient.TektonV1().Pipelines(ctx.env.GetAppNamespace()).Get(context.Background(), "pipeline1", metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, 1, len(pipeline.Spec.Tasks))
				task := pipeline.Spec.Tasks[0]
				assert.Equal(t, 1, len(task.TaskSpec.Sidecars))
				sidecar := task.TaskSpec.Sidecars[0]
				assert.NotNil(t, sidecar.SecurityContext)
				assert.Equal(t, commonUtils.BoolPtr(true), sidecar.SecurityContext.RunAsNonRoot)
				assert.Equal(t, commonUtils.BoolPtr(false), sidecar.SecurityContext.Privileged)
				assert.Equal(t, commonUtils.BoolPtr(false), sidecar.SecurityContext.AllowPrivilegeEscalation)
				assert.NotNil(t, sidecar.SecurityContext.Capabilities)
				assert.Equal(t, []corev1.Capability{"ALL"}, sidecar.SecurityContext.Capabilities.Drop)
			},
		},
		{
			name: "set SecurityContexts in task stepTemplate",
			fields: fields{
				targetEnvironments: map[string]bool{envDev: true},
			},
			args: args{envName: envDev, pipeline: getTestPipeline(func(pipeline *tekton.Pipeline) { pipeline.ObjectMeta.Name = "pipeline1" }),
				tasks: []tekton.Task{*getTestTask(func(task *tekton.Task) {
					task.Spec = tekton.TaskSpec{
						Steps: []tekton.Step{{Name: "step1"}},
					}
				})}, timestamp: "2020-01-01T00:00:00Z"},
			wantErr: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
			assertScenario: func(t *testing.T, ctx *pipelineContext) {
				pipeline, err := ctx.tektonClient.TektonV1().Pipelines(ctx.env.GetAppNamespace()).Get(context.Background(), "pipeline1", metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, 1, len(pipeline.Spec.Tasks))
				task := pipeline.Spec.Tasks[0]
				stepTemplate := task.TaskSpec.StepTemplate
				assert.NotNil(t, stepTemplate)
				assert.NotNil(t, stepTemplate.SecurityContext)
				assert.Equal(t, commonUtils.BoolPtr(true), stepTemplate.SecurityContext.RunAsNonRoot)
				assert.Equal(t, commonUtils.BoolPtr(false), stepTemplate.SecurityContext.Privileged)
				assert.Equal(t, commonUtils.BoolPtr(false), stepTemplate.SecurityContext.AllowPrivilegeEscalation)
				assert.NotNil(t, stepTemplate.SecurityContext.Capabilities)
				assert.Equal(t, []corev1.Capability{"ALL"}, stepTemplate.SecurityContext.Capabilities.Drop)
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ctx := &pipelineContext{
				radixClient:        radixclientfake.NewSimpleClientset(),
				kubeClient:         kubeclientfake.NewSimpleClientset(),
				tektonClient:       tektonclientfake.NewSimpleClientset(),
				env:                scenario.fields.env,
				radixApplication:   scenario.fields.radixApplication,
				targetEnvironments: scenario.fields.targetEnvironments,
				hash:               scenario.fields.hash,
				ownerReference:     scenario.fields.ownerReference,
			}
			if ctx.radixApplication == nil {
				ctx.radixApplication = getRadixApplication(appName, envDev, branchMain)
			}
			if ctx.env == nil {
				ctx.env = getEnvMock(mockCtrl, func(mockEnv *env.MockEnv) {
					mockEnv.EXPECT().GetAppName().Return(appName).AnyTimes()
					mockEnv.EXPECT().GetRadixImageTag().Return(radixImageTag).AnyTimes()
					mockEnv.EXPECT().GetRadixPipelineJobName().Return(radixPipelineJobName).AnyTimes()
					mockEnv.EXPECT().GetBranch().Return(branchMain).AnyTimes()
					mockEnv.EXPECT().GetAppNamespace().Return(utils.GetAppNamespace(appName)).AnyTimes()
				})
			}
			if ctx.ownerReference == nil {
				ctx.ownerReference = &metav1.OwnerReference{
					APIVersion: "radix.equinor.com/v1",
					Kind:       "RadixApplication",
					Name:       appName,
				}
			}
			if ctx.hash == "" {
				ctx.hash = hash
			}
			scenario.wantErr(t, ctx.createPipeline(scenario.args.envName, scenario.args.pipeline, scenario.args.tasks, scenario.args.timestamp))
		})
	}
}

func getEnvMock(mockCtrl *gomock.Controller, modify func(mockEnv *env.MockEnv)) *env.MockEnv {
	mockEnv := env.NewMockEnv(mockCtrl)
	if modify != nil {
		modify(mockEnv)
	}
	return mockEnv
}

func getTestPipeline(modify func(pipeline *tekton.Pipeline)) *tekton.Pipeline {
	task := &tekton.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "pipeline1"},
		Spec: tekton.PipelineSpec{
			Tasks: []tekton.PipelineTask{},
		},
	}
	if modify != nil {
		modify(task)
	}
	return task
}

func getRadixApplication(appName, environment, buildFrom string) *v1.RadixApplication {
	return utils.NewRadixApplicationBuilder().WithAppName(appName).
		WithEnvironment(environment, buildFrom).
		WithComponent(utils.NewApplicationComponentBuilder().WithName("comp1").WithPort("http", 8080).WithPublicPort("http")).
		BuildRA()
}

func getTestTask(modify func(task *tekton.Task)) *tekton.Task {
	task := &tekton.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task1"},
		Spec: tekton.TaskSpec{
			Steps: []tekton.Step{{Name: "step1"}},
		},
	}
	if modify != nil {
		modify(task)
	}
	return task
}
