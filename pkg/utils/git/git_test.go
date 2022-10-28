package git

import (
	"archive/zip"
	"fmt"
	"github.com/equinor/radix-tekton/pkg/utils/tests"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	_, err = os.Stat(unzipDestination)
	if err == nil {
		err := os.RemoveAll(unzipDestination)
		if err != nil {
			return err
		}
	}
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
	setupLog(t)
	gitDirPath := setupGitTest("test_data2.zip", "test_data2")

	releaseBranchHeadCommitHash := "a1ee44808de2a42d291b59fefb5c66b8ff6bf898"
	commitHash, err := GetGitCommitHashFromHead(gitDirPath, "this-branch-is-only-remote")
	assert.NoError(t, err)
	assert.Equal(t, commitHash, releaseBranchHeadCommitHash)

	tearDownGitTest()
}

func TestGetGitCommitTags(t *testing.T) {
	setupLog(t)
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

func TestGetGitChangedFolders_DummyRepo(t *testing.T) {
	setupLog(t)
	scenarios := []struct {
		name                      string
		beforeCommitExclusive     string
		targetCommit              string
		configFile                string
		configBranch              string
		expectedChangedFolders    []string
		expectedChangedConfigFile bool
		expectedError             string
	}{
		{
			name:                      "init - add radixconfig and gitignore files",
			targetCommit:              "7d6309f7537baa2815bb631802e6d8d613150c52",
			beforeCommitExclusive:     "",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"."},
			expectedChangedConfigFile: true,
		},
		//{
		//	name:                      "added app1 folder and its files. app1 component added to the radixconfig",
		//	targetCommit:              "0b9ee1f93639fff492c05b8d5e662301f508debe",
		//	beforeCommitExclusive:     "7d6309f7537baa2815bb631802e6d8d613150c52",
		//	configFile:                "radixconfig.yaml",
		//	configBranch:              "main",
		//	expectedChangedFolders:    []string{".", "app1"},
		//	expectedChangedConfigFile: true,
		//},
		//{
		//	name:                      "changed files in the folder app1",
		//	targetCommit:              "f68e88664ed51f79880b7f69d5789d21086ed1dc",
		//	beforeCommitExclusive:     "0b9ee1f93639fff492c05b8d5e662301f508debe",
		//	configFile:                "radixconfig.yaml",
		//	configBranch:              "main",
		//	expectedChangedFolders:    []string{"app1"},
		//	expectedChangedConfigFile: false,
		//},
		{
			name:                  "invalid the same target and before commit",
			targetCommit:          "7d6309f7537baa2815bb631802e6d8d613150c52",
			beforeCommitExclusive: "7d6309f7537baa2815bb631802e6d8d613150c52",
			configFile:            "radixconfig.yaml",
			configBranch:          "main",
			expectedError:         "beforeCommit cannot be equal to the targetCommit",
		},
		{
			name:                  "invalid target commit",
			targetCommit:          "invalid-commit",
			beforeCommitExclusive: "",
			configFile:            "radixconfig.yaml",
			configBranch:          "main",
			expectedError:         "invalid targetCommit",
		},
	}
	gitDirPath := "/Users/SSMOL/dev/go/src/github.com/equinor/test-data-git-commits"
	//gitDirPath := setupGitTest("test-data-git-commits.zip", "test-data-git-commits")
	for _, scenario := range scenarios {
		t.Logf("- test-case: %s", scenario.name)
		changedFolderList, changedConfigFile, err := GetGitAffectedResourcesBetweenCommits(gitDirPath, scenario.beforeCommitExclusive, scenario.targetCommit, scenario.configFile, scenario.configBranch)
		if scenario.expectedError == "" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Equal(t, scenario.expectedError, err.Error())
		}
		assert.ElementsMatch(t, scenario.expectedChangedFolders, changedFolderList)
		assert.Equal(t, scenario.expectedChangedConfigFile, changedConfigFile)
	}
	tearDownGitTest()
}

func setupLog(t *testing.T) {
	log.AddHook(tests.NewTestLogHook(t, log.DebugLevel).
		ModifyFormatter(func(f *log.TextFormatter) { f.DisableTimestamp = true }))
}
