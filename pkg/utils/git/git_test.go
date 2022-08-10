package git

import (
	"archive/zip"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const unzipDestination = "7c55c884-7a3e-4b1d-bb03-e7f8ce235d50"

func unzip(archivePath string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(unzipDestination, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(unzipDestination)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path")
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return err
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}

func getGitDir(testDataDir string) string {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	gitDirPath := fmt.Sprintf("%s/%s/%s/.git", workingDir, unzipDestination, testDataDir)
	_, err = os.Stat(gitDirPath)
	if err != nil {
		panic(err)
	}
	return gitDirPath
}

func TestGetGitCommitHashFromHead_DummyRepo(t *testing.T) {
	gitDirPath := setupGitTest("test_data.zip", "test_data")

	releaseBranchHeadCommitHash := "43332ef8f8a8c3830a235a5af7ac9098142e3af8"
	commitHash, err := GetGitCommitHashFromHead(gitDirPath, "release")
	assert.NoError(t, err)
	assert.Equal(t, commitHash, releaseBranchHeadCommitHash)

	tearDownGitTest()
}

func setupGitTest(testDataArchive, unzippedDir string) string {
	err := unzip(testDataArchive)
	if err != nil {
		panic(err)
	}
	return getGitDir(unzippedDir)
}

func tearDownGitTest() {
	err := os.RemoveAll(unzipDestination)
	if err != nil {
		panic(err)
	}
}

func TestGetGitCommitHashFromHead_DummyRepo2(t *testing.T) {
	gitDirPath := setupGitTest("test_data2.zip", "test_data2")

	releaseBranchHeadCommitHash := "a1ee44808de2a42d291b59fefb5c66b8ff6bf898"
	commitHash, err := GetGitCommitHashFromHead(gitDirPath, "this-branch-is-only-remote")
	assert.NoError(t, err)
	assert.Equal(t, commitHash, releaseBranchHeadCommitHash)

	tearDownGitTest()
}

func TestGetGitCommitTags(t *testing.T) {
	gitDirPath := setupGitTest("test_data.zip", "test_data")

	branchName := "branch-with-tags"
	tag0 := "special&%¤tag"
	tag1 := "v1.12"
	commitHash, err := GetGitCommitHashFromHead(gitDirPath, branchName)
	assert.NoError(t, err)
	tagsString, err := GetGitCommitTags(gitDirPath, commitHash)
	assert.NoError(t, err)
	tags := strings.Split(tagsString, " ")
	assert.Equal(t, tag0, tags[0])
	assert.Equal(t, tag1, tags[1])

	tearDownGitTest()
}
