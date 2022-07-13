package git

import (
	"encoding/hex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"strings"
)

func GetGitCommitHashAndTags(gitDir string) (string, string, error) {

	// Instantiate a new repository targeting the given path (the .git folder)
	r, err := git.PlainOpen(gitDir)
	if err != nil {
		return "", "", err
	}

	// Get HEAD reference to use for comparison
	ref, err := r.Head()
	if err != nil {
		return "", "", err
	}

	tags, err := r.Tags()
	if err != nil {
		return "", "", err
	}
	var tagNames []string

	// List all tags, both lightweight tags and annotated tags and see if some tag points to HEAD reference.
	err = tags.ForEach(func(t *plumbing.Reference) error {
		revHash, err := r.ResolveRevision(plumbing.Revision(t.Name()))
		if err != nil {
			return err
		}
		if *revHash == ref.Hash() {
			rawTagName := string(t.Name())
			tagName := parseTagName(rawTagName)
			tagNames = append(tagNames, tagName)
			//matchingTags += fmt.Sprintf(" %s", t.Name())
		}
		return nil
	})
	hashBytes := ref.Hash()
	return hex.EncodeToString(hashBytes[:]), strings.Join(tagNames, " "), nil
}

func parseTagName(rawTagName string) string { //"refs/tags/tag1"
	prefixToRemove := "refs/tags/"
	if rawTagName[:len(prefixToRemove)] == prefixToRemove {
		return rawTagName[len(prefixToRemove):]
	}
	return rawTagName
}
