package git

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type lastEnvironmentDeployCommit struct {
	envName    string
	commitHash string
}

// GetCommitHashAndTags gets target commit hash and tags from GitHub repository
func GetCommitHashAndTags(env env.Env) (string, string, error) {
	gitWorkspace := env.GetGitRepositoryWorkspace()
	targetCommitHash, err := getGitCommitHash(gitWorkspace, env)
	if err != nil {
		return "", "", err
	}

	gitTags, err := getGitCommitTags(getGitDir(gitWorkspace), targetCommitHash)
	if err != nil {
		return "", "", err
	}
	return targetCommitHash, gitTags, nil
}

func getGitDir(gitWorkspace string) string {
	return gitWorkspace + "/.git"
}

// getGitCommitHashFromHead returns the commit hash for the HEAD of branchName in gitDir
func getGitCommitHashFromHead(gitDir string, branchName string) (string, error) {

	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return "", err
	}
	log.Debugf("opened gitDir %s", gitDir)

	// Get branchName hash
	commitHash, err := getBranchCommitHash(r, branchName)
	if err != nil {
		return "", err
	}
	log.Debugf("resolved branch %s", branchName)

	hashBytesString := hex.EncodeToString(commitHash[:])
	return hashBytesString, nil
}

// getGitAffectedResourcesBetweenCommits returns the list of folders, where files were affected after beforeCommitHash (not included) till targetCommitHash commit (included)
func getGitAffectedResourcesBetweenCommits(gitDir, targetCommitString, beforeCommitString, configFile, configBranch string) ([]string, bool, error) {
	targetCommitHash, err := getTargetCommitHash(beforeCommitString, targetCommitString)
	if err != nil {
		return nil, false, err
	}
	repository, currentBranch, err := getRepository(gitDir)
	if err != nil {
		return nil, false, err
	}
	beforeCommitHash, err := getBeforeCommitHash(beforeCommitString, repository)
	if (err != nil && err != io.EOF) && beforeCommitHash == nil {
		return nil, false, err
	}
	beforeCommit, err := repository.CommitObject(*beforeCommitHash)
	if err != nil {
		return nil, false, err
	}

	if strings.EqualFold(beforeCommitHash.String(), targetCommitString) { //targetCommit is the very first commit in the repo
		return getChangedFoldersOfCommitFiles(beforeCommit, configBranch, currentBranch, configFile)
	}

	targetCommit, err := repository.CommitObject(*targetCommitHash)
	if (err != nil && err != io.EOF) && targetCommit == nil {
		return nil, false, err
	}
	return getChangedFoldersFromTargetCommitTillExclusiveBeforeCommit(beforeCommit, targetCommit, configBranch, currentBranch, configFile)
}

func getChangedFoldersFromTargetCommitTillExclusiveBeforeCommit(targetCommit *object.Commit, beforeCommit *object.Commit, configBranch string, currentBranch string, configFile string) ([]string, bool, error) {
	patch, err := beforeCommit.Patch(targetCommit)
	if err != nil {
		return nil, false, err
	}
	changedFolderNamesMap := make(map[string]bool)
	changedConfigFile := false
	for _, filePatch := range patch.FilePatches() {
		fromFile, toFile := filePatch.Files()
		for _, file := range []diff.File{fromFile, toFile} {
			if file != nil {
				appendFolderToMap(changedFolderNamesMap, &changedConfigFile, configBranch, currentBranch, configFile, file.Path(), file.Mode())
			}
		}
	}
	return maps.GetKeysFromMap(changedFolderNamesMap), changedConfigFile, nil
}

func getChangedFoldersOfCommitFiles(commit *object.Commit, configBranch string, currentBranch string, configFile string) ([]string, bool, error) {
	changedFolderNamesMap := make(map[string]bool)
	changedConfigFile := false
	fileIter, err := commit.Files()
	if err != nil {
		return nil, false, err
	}
	err = fileIter.ForEach(func(file *object.File) error {
		appendFolderToMap(changedFolderNamesMap, &changedConfigFile, configBranch, currentBranch, configFile, file.Name, file.Mode)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return maps.GetKeysFromMap(changedFolderNamesMap), changedConfigFile, nil
}

func getRepository(gitDir string) (*git.Repository, string, error) {
	log.Debugf("opened gitDir %s", gitDir)
	repository, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, "", err
	}
	currentBranch, err := getCurrentBranch(repository)
	if err != nil {
		return nil, "", err
	}
	return repository, currentBranch, nil
}

func getTargetCommitHash(beforeCommitString, targetCommitString string) (*plumbing.Hash, error) {
	targetCommitHash := plumbing.NewHash(targetCommitString)
	if targetCommitHash == plumbing.ZeroHash {
		return nil, errors.New("invalid targetCommit")
	}
	if strings.EqualFold(beforeCommitString, targetCommitString) {
		return nil, errors.New("beforeCommit cannot be equal to the targetCommit")
	}
	return &targetCommitHash, nil
}

func getCurrentBranch(repository *git.Repository) (string, error) {
	head, err := repository.Head()
	if err != nil {
		return "", err
	}
	branchHeadNamePrefix := "refs/heads/"
	branchHeadName := head.Name().String()
	if head.Name() == "HEAD" || !strings.HasPrefix(branchHeadName, branchHeadNamePrefix) {
		return "", errors.New("unexpected current git revision")
	}
	currentBranch := strings.TrimPrefix(branchHeadName, branchHeadNamePrefix)
	return currentBranch, nil
}

func appendFolderToMap(changedFolderNamesMap map[string]bool, changedConfigFile *bool, configBranch string, currentBranch string, configFile string, filePath string, fileMode filemode.FileMode) {
	if filePath == "" {
		return
	}
	folderName := ""
	if fileMode == filemode.Dir {
		folderName = filePath
	} else {
		folderName = filepath.Dir(filePath)
		if !*changedConfigFile && strings.EqualFold(configBranch, currentBranch) && strings.EqualFold(configFile, filePath) {
			*changedConfigFile = true
		}
		log.Debugf("- file: %s", filePath)
	}
	if _, ok := changedFolderNamesMap[folderName]; !ok {
		changedFolderNamesMap[folderName] = true
	}
}

func getBeforeCommitHash(commitHash string, repository *git.Repository) (*plumbing.Hash, error) {
	logIter, err := repository.Log(&git.LogOptions{
		Order: git.LogOrderBSF,
	})
	if err != nil {
		return nil, err
	}
	var hash plumbing.Hash
	err = logIter.ForEach(func(c *object.Commit) error {
		hash = c.Hash
		if len(commitHash) > 0 && c.Hash.String() == commitHash {
			return io.EOF
		}
		return nil
	})
	return &hash, err
}

func getBranchCommitHash(r *git.Repository, branchName string) (*plumbing.Hash, error) {
	// first, we try to resolve a local revision. If possible, this is best. This succeeds if code branch and config
	// branch are the same
	commitHash, err := r.ResolveRevision(plumbing.Revision(branchName))
	if err != nil {
		// on second try, we try to resolve the remote branch. This introduces a chance that the remote has been altered
		// with new hash after initial clone
		commitHash, err = r.ResolveRevision(plumbing.Revision(fmt.Sprintf("refs/remotes/origin/%s", branchName)))
		if err != nil {
			if strings.EqualFold(err.Error(), "reference not found") {
				return nil, errors.New(fmt.Sprintf("there is no branch %s or access to the repository", branchName))
			}
			return nil, err
		}
	}
	return commitHash, nil
}

// getGitCommitTags returns any git tags which point to commitHash
func getGitCommitTags(gitDir string, commitHashString string) (string, error) {

	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return "", err
	}

	commitHash := plumbing.NewHash(commitHashString)

	tags, err := r.Tags()
	if err != nil {
		return "", err
	}
	var tagNames []string

	// List all tags, both lightweight tags and annotated tags and see if any tags point to HEAD reference.
	err = tags.ForEach(func(t *plumbing.Reference) error {
		revHash, err := r.ResolveRevision(plumbing.Revision(t.Name()))
		if err != nil {
			return err
		}
		if *revHash == commitHash {
			rawTagName := string(t.Name())
			tagName := parseTagName(rawTagName)
			tagNames = append(tagNames, tagName)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	tagNamesString := strings.Join(tagNames, " ")

	return tagNamesString, nil
}

func parseTagName(rawTagName string) string {
	prefixToRemove := "refs/tags/"
	if rawTagName[:len(prefixToRemove)] == prefixToRemove {
		return rawTagName[len(prefixToRemove):]
	}
	return rawTagName // this line is expected to never be executed
}

func getGitCommitHash(workspace string, e env.Env) (string, error) {
	webhookCommitId := e.GetWebhookCommitId()
	if webhookCommitId != "" {
		log.Debugf("got git commit hash %s from env var %s", webhookCommitId, defaults.RadixGithubWebhookCommitId)
		return webhookCommitId, nil
	}
	branchName := e.GetBranch()
	log.Debugf("determining git commit hash of HEAD of branch %s", branchName)
	gitCommitHash, err := getGitCommitHashFromHead(workspace+"/.git", branchName)
	log.Debugf("got git commit hash %s from HEAD of branch %s", gitCommitHash, branchName)
	return gitCommitHash, err
}

// GetChangesFromGitRepository Get changed folders in environments and if radixconfig.yaml was changed
func GetChangesFromGitRepository(radixClient radixclient.Interface, appName string, environments []string, radixConfigFileName, gitWorkspace, radixConfigBranch, targetCommitHash string) (map[string][]string, bool, error) {
	envChanges := make(map[string][]string, len(environments))
	radixConfigWasChanged := false
	beforeCommitHash, err := getLastSuccessfulEnvironmentDeployCommits(radixClient, appName, environments)
	if err != nil {
		return nil, false, err
	}
	for envName, beforeCommitHash := range beforeCommitHash {
		changedFolders, radixConfigChanged, err := getGitAffectedResourcesBetweenCommits(getGitDir(gitWorkspace), targetCommitHash, beforeCommitHash, radixConfigFileName, radixConfigBranch)
		envChanges[envName] = changedFolders
		if err != nil {
			return nil, false, err
		}
		radixConfigWasChanged = radixConfigWasChanged || radixConfigChanged

	}
	return envChanges, radixConfigWasChanged, nil
}

func getLastSuccessfulEnvironmentDeployCommits(radixClient radixclient.Interface, appName string, environments []string) (map[string]string, error) {
	envCommitMap := make(map[string]string)
	appNamespace := utils.GetAppNamespace(appName)
	jobTypeMap, err := getJobTypeMap(radixClient, appNamespace)
	if err != nil {
		return nil, err
	}
	for _, envName := range environments {
		radixDeployments, err := getRadixDeployments(radixClient, appName, envName)
		if err != nil {
			return nil, err
		}
		envCommitMap[envName] = getLastRadixDeploymentCommitHash(radixDeployments, jobTypeMap)
	}
	return envCommitMap, nil
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

func getRadixDeployments(radixClient radixclient.Interface, appName, envName string) ([]v1.RadixDeployment, error) {
	namespace := utils.GetEnvironmentNamespace(appName, envName)
	deployments := radixClient.RadixV1().RadixDeployments(namespace)
	radixDeploymentList, err := deployments.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return radixDeploymentList.Items, nil
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
