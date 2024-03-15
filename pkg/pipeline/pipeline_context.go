package pipeline

import (
	"fmt"
	"strings"

	"github.com/equinor/radix-tekton/pkg/internal/wait"
	"github.com/equinor/radix-tekton/pkg/utils/git"
	"github.com/equinor/radix-tekton/pkg/utils/radix/applicationconfig"

	"github.com/equinor/radix-common/utils"
	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/models/env"
	ownerreferences "github.com/equinor/radix-tekton/pkg/utils/owner_references"
	"github.com/rs/zerolog/log"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type pipelineContext struct {
	radixClient        radixclient.Interface
	kubeClient         kubernetes.Interface
	tektonClient       tektonclient.Interface
	env                env.Env
	radixApplication   *v1.RadixApplication
	targetEnvironments map[string]bool
	hash               string
	ownerReference     *metav1.OwnerReference
	waiter             wait.PipelineRunsCompletionWaiter
}

func (ctx *pipelineContext) GetEnv() env.Env {
	return ctx.env
}

func (ctx *pipelineContext) GetHash() string {
	return ctx.hash
}

func (ctx *pipelineContext) GetKubeClient() kubernetes.Interface {
	return ctx.kubeClient
}

func (ctx *pipelineContext) GetTektonClient() tektonclient.Interface {
	return ctx.tektonClient
}

func (ctx *pipelineContext) GetRadixApplication() *v1.RadixApplication {
	return ctx.radixApplication
}

func (ctx *pipelineContext) GetPipelineRunsWaiter() wait.PipelineRunsCompletionWaiter {
	return ctx.waiter
}

func (ctx *pipelineContext) getEnvVars(targetEnv string) v1.EnvVarsMap {
	envVarsMap := make(v1.EnvVarsMap)
	ctx.setPipelineRunParamsFromBuild(envVarsMap)
	ctx.setPipelineRunParamsFromEnvironmentBuilds(targetEnv, envVarsMap)
	return envVarsMap
}

func (ctx *pipelineContext) setPipelineRunParamsFromBuild(envVarsMap v1.EnvVarsMap) {
	if ctx.radixApplication.Spec.Build == nil ||
		ctx.radixApplication.Spec.Build.Variables == nil ||
		len(ctx.radixApplication.Spec.Build.Variables) == 0 {
		log.Debug().Msg("No radixApplication build variables")
		return
	}

	for name, envVar := range ctx.radixApplication.Spec.Build.Variables {
		envVarsMap[name] = envVar
	}
}

func (ctx *pipelineContext) setPipelineRunParamsFromEnvironmentBuilds(targetEnv string, envVarsMap v1.EnvVarsMap) {
	for _, buildEnv := range ctx.radixApplication.Spec.Environments {
		if !strings.EqualFold(buildEnv.Name, targetEnv) || buildEnv.Build.Variables == nil {
			continue
		}
		for envVarName, envVarVal := range buildEnv.Build.Variables {
			envVarsMap[envVarName] = envVarVal // Overrides common env-vars from Spec.Build, if any
		}
	}
}

// getGitHash return git commit to which the user repository should be reset before parsing sub-pipelines.
func (ctx *pipelineContext) getGitHash() (string, error) {
	if ctx.env.GetRadixPipelineType() == v1.Build {
		log.Info().Msg("build job with no deployment, skipping sub-pipelines.")
		return "", nil
	}

	if ctx.env.GetRadixPipelineType() == v1.Promote {
		sourceRdHashFromAnnotation := ctx.env.GetSourceDeploymentGitCommitHash()
		sourceDeploymentGitBranch := ctx.env.GetSourceDeploymentGitBranch()
		if sourceRdHashFromAnnotation != "" {
			return sourceRdHashFromAnnotation, nil
		}
		if sourceDeploymentGitBranch == "" {
			log.Info().Msg("source deployment has no git metadata, skipping sub-pipelines")
			return "", nil
		}
		sourceRdHashFromBranchHead, err := git.GetGitCommitHashFromHead(ctx.env.GetGitRepositoryWorkspace(), sourceDeploymentGitBranch)
		if err != nil {
			return "", nil
		}
		return sourceRdHashFromBranchHead, nil
	}

	if ctx.env.GetRadixPipelineType() == v1.Deploy {
		pipelineJobBranch := ""
		re := applicationconfig.GetEnvironmentFromRadixApplication(ctx.radixApplication, ctx.env.GetRadixDeployToEnvironment())
		if re != nil {
			pipelineJobBranch = re.Build.From
		}
		if pipelineJobBranch == "" {
			log.Info().Msg("deploy job with no build branch, skipping sub-pipelines.")
			return "", nil
		}
		gitHash, err := git.GetGitCommitHashFromHead(ctx.env.GetGitRepositoryWorkspace(), pipelineJobBranch)
		if err != nil {
			return "", err
		}
		return gitHash, nil
	}

	if ctx.env.GetRadixPipelineType() == v1.BuildDeploy {
		gitHash, err := git.GetGitCommitHash(ctx.env.GetGitRepositoryWorkspace(), ctx.env)
		if err != nil {
			return "", err
		}
		return gitHash, nil
	}
	return "", fmt.Errorf("unknown pipeline type %s", ctx.env.GetRadixPipelineType())
}

type NewPipelineContextOption func(ctx *pipelineContext)

// NewPipelineContext Create new NewPipelineContext instance
func NewPipelineContext(kubeClient kubernetes.Interface, radixClient radixclient.Interface, tektonClient tektonclient.Interface, environment env.Env, options ...NewPipelineContextOption) models.Context {
	ownerReference := ownerreferences.GetOwnerReferenceOfJobFromLabels()
	ctx := &pipelineContext{
		kubeClient:     kubeClient,
		radixClient:    radixClient,
		tektonClient:   tektonClient,
		env:            environment,
		hash:           strings.ToLower(utils.RandStringStrSeed(5, environment.GetRadixPipelineJobName())),
		ownerReference: ownerReference,
		waiter:         wait.NewPipelineRunsCompletionWaiter(tektonClient),
	}

	for _, option := range options {
		option(ctx)
	}

	return ctx
}

func WithPipelineRunsWaiter(waiter wait.PipelineRunsCompletionWaiter) NewPipelineContextOption {
	return func(ctx *pipelineContext) {
		ctx.waiter = waiter
	}
}
