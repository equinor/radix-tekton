package git

import (
	"encoding/hex"
	"fmt"
	"github.com/equinor/radix-common/utils/maps"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

// GetGitAffectedResourcesBetweenCommits returns the list of folders, where files were affected between commits
func GetGitAffectedResourcesBetweenCommits(gitDir, olderCommit, newerCommit, configFile, configBranch string) ([]string, bool, error) {
	log.Debug("test")
	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, false, err
	}
	log.Debugf("opened gitDir %s", gitDir)

	logOptions := git.LogOptions{
		Order: git.LogOrderCommitterTime,
	}
	if len(newerCommit) > 0 {
		logOptions.From = plumbing.NewHash(newerCommit)
	}
	w, err := r.Worktree()
	if err != nil {
		return nil, false, err
	}
	fmt.Println(w)

	logIter, err := r.Log(&logOptions)
	if err != nil {
		return nil, false, err
	}

	changedFolderNamesMap := make(map[string]bool)
	changedConfigFile := false
	err = logIter.ForEach(func(c *object.Commit) error {
		fmt.Println(c.Hash)
		files, err := c.Files()
		if err != nil {
			return err
		}
		commitHash := c.Hash.String()
		log.Debugf("changes in the commit: %s", commitHash)
		files.ForEach(func(file *object.File) error {
			name := ""
			if file.Mode == filemode.Dir {
				name = file.Name
			} else {
				name = filepath.Dir(file.Name)
				if !changedConfigFile && strings.EqualFold(configFile, file.Name) {
					changedConfigFile = true
				}
				log.Debugf("- file: %s", file.Name)
			}
			if _, ok := changedFolderNamesMap[name]; !ok {
				changedFolderNamesMap[name] = true
				log.Debugf("- dir: %s", name)
			}
			return nil
		})
		if len(olderCommit) > 0 && commitHash == olderCommit {
			return io.EOF
		}
		return nil
	})

	return maps.GetKeysFromMap(changedFolderNamesMap), changedConfigFile, nil
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
