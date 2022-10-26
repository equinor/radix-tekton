package git

import (
	"encoding/hex"
	"fmt"
	"github.com/equinor/radix-common/utils/maps"
	"github.com/go-git/go-git/v5/plumbing/object"
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

// GetGitAffectedFilesAndFoldersBetweenCommits returns the list of files and folders, affected between commits
func GetGitAffectedFilesAndFoldersBetweenCommits(gitDir, beginCommit, endCommit string) ([]string, error) {

	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return nil, err
	}
	log.Debugf("opened gitDir %s", gitDir)

	cIter, err := r.Log(&git.LogOptions{
		From:  plumbing.NewHash(beginCommit),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}

	fileNamesMap := make(map[string]bool)
	// ... just iterates over the commits, printing it
	err = cIter.ForEach(func(c *object.Commit) error {
		fmt.Println(c.Hash)
		files, err := c.Files()
		if err != nil {
			return err
		}
		files.ForEach(func(file *object.File) error {
			fileNamesMap[file.Name] = true
			return nil
		})
		if c.Hash.String() == endCommit {
			cIter.Close()
		}
		return nil
	})

	return maps.GetKeysFromMap(fileNamesMap), nil
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
