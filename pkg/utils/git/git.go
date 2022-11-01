package git

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/equinor/radix-common/utils/maps"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	log "github.com/sirupsen/logrus"
)

// GetGitCommitHashFromHead returns the commit hash for the HEAD of branchName in gitDir
func GetGitCommitHashFromHead(gitDir string, branchName string) (string, error) {

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

// GetGitAffectedResourcesBetweenCommits returns the list of folders, where files were affected after beforeCommitHash (not included) till targetCommitHash commit (included)
func GetGitAffectedResourcesBetweenCommits(gitDir, targetCommitString, beforeCommitString, configFile, configBranch string) ([]string, bool, error) {
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
	fileIter.ForEach(func(file *object.File) error {
		appendFolderToMap(changedFolderNamesMap, &changedConfigFile, configBranch, currentBranch, configFile, file.Name, file.Mode)
		return nil
	})
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
			return nil, err
		}
	}
	return commitHash, nil
}

// GetGitCommitTags returns any git tags which point to commitHash
func GetGitCommitTags(gitDir string, commitHashString string) (string, error) {

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
