package git

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/utils/test"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func getTestGitDir(testDataDir string) string {
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
	commitHash, err := getGitCommitHashFromHead(gitDirPath, "release")
	assert.NoError(t, err)
	assert.Equal(t, commitHash, releaseBranchHeadCommitHash)

	tearDownGitTest()
}

func setupGitTest(testDataArchive, unzippedDir string) string {
	err := unzip(testDataArchive)
	if err != nil {
		panic(err)
	}
	return getTestGitDir(unzippedDir)
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
	commitHash, err := getGitCommitHashFromHead(gitDirPath, "this-branch-is-only-remote")
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
	commitHash, err := getGitCommitHashFromHead(gitDirPath, branchName)
	assert.NoError(t, err)
	tagsString, err := getGitCommitTags(gitDirPath, commitHash)
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
		{
			name:                      "added app1 folder and its files. app1 component added to the radixconfig",
			targetCommit:              "0b9ee1f93639fff492c05b8d5e662301f508debe",
			beforeCommitExclusive:     "7d6309f7537baa2815bb631802e6d8d613150c52",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{".", "app1"},
			expectedChangedConfigFile: true,
		},
		{
			name:                      "Changed radixconfig, but config branch is different",
			targetCommit:              "0b9ee1f93639fff492c05b8d5e662301f508debe",
			beforeCommitExclusive:     "7d6309f7537baa2815bb631802e6d8d613150c52",
			configFile:                "radixconfig.yaml",
			configBranch:              "another-branch",
			expectedChangedFolders:    []string{".", "app1"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "changed files in the folder app1",
			targetCommit:              "f68e88664ed51f79880b7f69d5789d21086ed1dc",
			beforeCommitExclusive:     "0b9ee1f93639fff492c05b8d5e662301f508debe",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app1"},
			expectedChangedConfigFile: false,
		},
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
		{
			name:                  "invalid empty target commit",
			targetCommit:          "",
			beforeCommitExclusive: "",
			configFile:            "radixconfig.yaml",
			configBranch:          "main",
			expectedError:         "invalid targetCommit",
		},
		{
			name:                      "Added folder app2 with files",
			targetCommit:              "157014b59d6b24205b4fbf57165f0029c49d7963",
			beforeCommitExclusive:     "f68e88664ed51f79880b7f69d5789d21086ed1dc",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app2"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "Folder app2 was renamed to app3, file in folder 3 was changed",
			targetCommit:              "31472b8fc3fe22a2b8d174e79ae2f891a975864d",
			beforeCommitExclusive:     "157014b59d6b24205b4fbf57165f0029c49d7963",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app2", "app3"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "File in folder 3 was changed",
			targetCommit:              "31472b8fc3fe22a2b8d174e79ae2f891a975864d",
			beforeCommitExclusive:     "8d99f5318a7164ff1e03bd4fbe1cba554e9c906b",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "File in folder 3 was changed",
			targetCommit:              "31472b8fc3fe22a2b8d174e79ae2f891a975864d",
			beforeCommitExclusive:     "8d99f5318a7164ff1e03bd4fbe1cba554e9c906b",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "radixconfig.yaml was added to the folder app3",
			targetCommit:              "d79cc7f32f58e30b01671b8bcc19f41508db95c8",
			beforeCommitExclusive:     "31472b8fc3fe22a2b8d174e79ae2f891a975864d",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "radixconfig.yaml was added to the folder app3, it is current config",
			targetCommit:              "d79cc7f32f58e30b01671b8bcc19f41508db95c8",
			beforeCommitExclusive:     "31472b8fc3fe22a2b8d174e79ae2f891a975864d",
			configFile:                "app3/radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: true,
		},
		{
			name:                      "radixconfig.yaml was changed to the folder app3",
			targetCommit:              "0143afb54d8f9e5451b2dc34f47d7f933080e7a4",
			beforeCommitExclusive:     "d79cc7f32f58e30b01671b8bcc19f41508db95c8",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "radixconfig.yaml was changed to the folder app3, it is current config",
			targetCommit:              "0143afb54d8f9e5451b2dc34f47d7f933080e7a4",
			beforeCommitExclusive:     "d79cc7f32f58e30b01671b8bcc19f41508db95c8",
			configFile:                "app3/radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3"},
			expectedChangedConfigFile: true,
		},
		{
			name:                      "Sub-folders were added to the folder app3, with file. Sub-folder was added to the folder app1, without file.",
			targetCommit:              "13bd8316267d6a0be44f3f87fe49e807e7bc24b4",
			beforeCommitExclusive:     "0143afb54d8f9e5451b2dc34f47d7f933080e7a4",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3/data/level2"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "Files added and changed in the sub-folders of app3. File was added to the sub-folder in the folder app1.",
			targetCommit:              "986065b74c8e9e4012287fdd6b13021591ce00c3",
			beforeCommitExclusive:     "13bd8316267d6a0be44f3f87fe49e807e7bc24b4",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3", "app3/data", "app3/data/level2", "app1/data"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "radixconfig-app3-level2.yaml was added to the subfolder of the folder app3",
			targetCommit:              "e89ac3d3ba66498cf6165e119d29a86b6b8183ab",
			beforeCommitExclusive:     "986065b74c8e9e4012287fdd6b13021591ce00c3",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3/data/level2"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "radixconfig-app3-level2.yaml was added to the subfolder of the folder app3, it is current config",
			targetCommit:              "e89ac3d3ba66498cf6165e119d29a86b6b8183ab",
			beforeCommitExclusive:     "986065b74c8e9e4012287fdd6b13021591ce00c3",
			configFile:                "app3/data/level2/radixconfig-app3-level2.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app3/data/level2"},
			expectedChangedConfigFile: true,
		},
		{
			name:                      "radixconfig-app3-level2.yaml was added to the subfolder of the folder app3, it is current config, but not this branch",
			targetCommit:              "e89ac3d3ba66498cf6165e119d29a86b6b8183ab",
			beforeCommitExclusive:     "986065b74c8e9e4012287fdd6b13021591ce00c3",
			configFile:                "app3/data/level2/radixconfig-app3-level2.yaml",
			configBranch:              "another-branch",
			expectedChangedFolders:    []string{"app3/data/level2"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "Files were changed in subfolders app1 and app3, with config radixconfig.yaml",
			targetCommit:              "2127927fa21ae471baefbadd3f05b60a4bf38b5f",
			beforeCommitExclusive:     "e89ac3d3ba66498cf6165e119d29a86b6b8183ab",
			configFile:                "radixconfig.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app1/data", "app3/data/level2"},
			expectedChangedConfigFile: false,
		},
		{
			name:                      "Files were changed in subfolders app1 and app3, with config app3/data/level2/radixconfig-app3-level2.yaml",
			targetCommit:              "2127927fa21ae471baefbadd3f05b60a4bf38b5f",
			beforeCommitExclusive:     "e89ac3d3ba66498cf6165e119d29a86b6b8183ab",
			configFile:                "app3/data/level2/radixconfig-app3-level2.yaml",
			configBranch:              "main",
			expectedChangedFolders:    []string{"app1/data", "app3/data/level2"},
			expectedChangedConfigFile: false,
		},
	}
	gitDirPath := "/users/SSMOL/dev/go/src/github.com/equinor/test-data-git-commits"
	//gitDirPath := setupGitTest("test-data-git-commits.zip", "test-data-git-commits")
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			changedFolderList, changedConfigFile, err := getGitAffectedResourcesBetweenCommits(gitDirPath, scenario.targetCommit, scenario.beforeCommitExclusive, scenario.configFile, scenario.configBranch)
			if scenario.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, scenario.expectedError, err.Error())
			}
			assert.ElementsMatch(t, scenario.expectedChangedFolders, changedFolderList)
			assert.Equal(t, scenario.expectedChangedConfigFile, changedConfigFile)
		})
	}
	tearDownGitTest()
}

type scenarioGetLastSuccessfulEnvironmentDeployCommits struct {
	name                       string
	environments               []string
	envRadixDeploymentBuilders map[string][]utils.DeploymentBuilder
	expectedEnvCommits         map[string]string
	pipelineJobs               []*v1.RadixJob
}

func TestGetLastSuccessfulEnvironmentDeployCommits(t *testing.T) {
	const appName = "anApp"
	timeNow := time.Now()
	scenarios := []scenarioGetLastSuccessfulEnvironmentDeployCommits{
		{
			name:         "no deployments - no commits",
			environments: []string{"dev", "qa"},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": nil,
			},
			expectedEnvCommits: map[string]string{"dev": "", "qa": ""},
		},
		{
			name:         "deployment has no radix-job - no commit",
			environments: []string{"dev", "qa"},
			pipelineJobs: nil,
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1"),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "", "qa": ""},
		},
		{
			name:         "one deployment with commit in annotation - its commit",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1"),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit1", "qa": ""},
		},
		{
			name:         "one deployment with commit in label - its commit",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithLabel(kube.RadixCommitLabel, "commit1").
						WithLabel(kube.RadixJobNameLabel, "job1"),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit1", "qa": ""},
		},
		{
			name:         "one deployment with commit both in label and annotation - takes commit from annotation",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit-in-annotation",
						}).
						WithLabel(kube.RadixCommitLabel, "commit-in-label").
						WithLabel(kube.RadixJobNameLabel, "job1"),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit-in-annotation", "qa": ""},
		},
		{
			name:         "multiple deployments - gets the commit from the latest deployment",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
				createPipelineJobs(appName, "job2", v1.BuildDeploy),
				createPipelineJobs(appName, "job3", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCreated(timeNow.Add(time.Hour * time.Duration(1))).
						WithCondition(v1.DeploymentInactive),
					getRadixDeploymentBuilderFor("dev", "rd3").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit3",
						}).
						WithLabel(kube.RadixJobNameLabel, "job3").
						WithCreated(timeNow.Add(time.Hour * time.Duration(9))).
						WithCondition(v1.DeploymentActive),
					getRadixDeploymentBuilderFor("dev", "rd2").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit2",
						}).
						WithLabel(kube.RadixJobNameLabel, "job2").
						WithCreated(timeNow.Add(time.Hour * time.Duration(5))).
						WithCondition(v1.DeploymentInactive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit3", "qa": ""},
		},
		{
			name:         "multiple deployments - not affected by status active",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
				createPipelineJobs(appName, "job2", v1.BuildDeploy),
				createPipelineJobs(appName, "job3", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCreated(timeNow.Add(time.Hour * time.Duration(1))).
						WithCondition(v1.DeploymentInactive),
					getRadixDeploymentBuilderFor("dev", "rd3").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit3",
						}).
						WithLabel(kube.RadixJobNameLabel, "job3").
						WithCreated(timeNow.Add(time.Hour * time.Duration(9))).
						WithCondition(v1.DeploymentInactive),
					getRadixDeploymentBuilderFor("dev", "rd2").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit2",
						}).
						WithLabel(kube.RadixJobNameLabel, "job2").
						WithCreated(timeNow.Add(time.Hour * time.Duration(5))).
						WithCondition(v1.DeploymentActive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit3", "qa": ""},
		},
		{
			name:         "multiple envs deployments - get env-specific commit",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
				createPipelineJobs(appName, "job2", v1.BuildDeploy),
				createPipelineJobs(appName, "job3", v1.BuildDeploy),
				createPipelineJobs(appName, "job4", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd2-qa").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit2-dev",
						}).
						WithLabel(kube.RadixJobNameLabel, "job2").
						WithCreated(timeNow.Add(time.Hour * time.Duration(5))).
						WithCondition(v1.DeploymentActive),
					getRadixDeploymentBuilderFor("dev", "rd1-qa").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1-dev",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCreated(timeNow.Add(time.Hour * time.Duration(1))).
						WithCondition(v1.DeploymentInactive),
				},
				"qa": {
					getRadixDeploymentBuilderFor("qa", "rd2-qa").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit2-qa",
						}).
						WithLabel(kube.RadixJobNameLabel, "job4").
						WithCreated(timeNow.Add(time.Hour * time.Duration(5))).
						WithCondition(v1.DeploymentActive),
					getRadixDeploymentBuilderFor("qa", "rd1-qa").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1-dev",
						}).
						WithLabel(kube.RadixJobNameLabel, "job3").
						WithCreated(timeNow.Add(time.Hour * time.Duration(1))).
						WithCondition(v1.DeploymentInactive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit2-dev", "qa": "commit2-qa"},
		},
		{
			name:         "multiple deployments - gets commit for build-deploy pipeline job radix deployment",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.BuildDeploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCondition(v1.DeploymentActive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "commit1", "qa": ""},
		},
		{
			name:         "multiple deployments - does not get commit for build pipeline job radix deployment",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.Build),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCondition(v1.DeploymentActive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "", "qa": ""},
		},
		{
			name:         "multiple deployments - does not get commit for deploy pipeline job radix deployment",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.Deploy),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCondition(v1.DeploymentActive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "", "qa": ""},
		},
		{
			name:         "multiple deployments - does not get commit for promote pipeline job radix deployment",
			environments: []string{"dev", "qa"},
			pipelineJobs: []*v1.RadixJob{
				createPipelineJobs(appName, "job1", v1.Promote),
			},
			envRadixDeploymentBuilders: map[string][]utils.DeploymentBuilder{
				"dev": {
					getRadixDeploymentBuilderFor("dev", "rd1").
						WithAnnotations(map[string]string{
							kube.RadixCommitAnnotation: "commit1",
						}).
						WithLabel(kube.RadixJobNameLabel, "job1").
						WithCondition(v1.DeploymentActive),
				},
			},
			expectedEnvCommits: map[string]string{"dev": "", "qa": ""},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			radixClient := prepareTestGetLastSuccessfulEnvironmentDeployCommits(t, appName, scenario)

			commits, err := getLastSuccessfulEnvironmentDeployCommits(radixClient, appName, scenario.environments)

			assert.NoError(t, err)
			commitEnvs := maps.GetKeysFromMap(commits)
			assert.ElementsMatch(t, scenario.environments, commitEnvs)
			for _, env := range commitEnvs {
				assert.Equal(t, scenario.expectedEnvCommits[env], commits[env])
			}
		})
	}
}

func getRadixDeploymentBuilderFor(env string, deploymentName string) utils.DeploymentBuilder {
	return utils.ARadixDeployment().
		WithEnvironment(env).
		WithDeploymentName(deploymentName)
}

func createPipelineJobs(appName, jobName string, pipelineType v1.RadixPipelineType) *v1.RadixJob {
	return &v1.RadixJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: utils.GetAppNamespace(appName),
		},
		Spec: v1.RadixJobSpec{
			PipeLineType: pipelineType,
		},
	}
}

func prepareTestGetLastSuccessfulEnvironmentDeployCommits(t *testing.T, appName string, scenario scenarioGetLastSuccessfulEnvironmentDeployCommits) radixclient.Interface {
	radixClient, _ := test.Setup(t)
	for _, environment := range scenario.environments {
		if radixDeploymentBuilders, ok := scenario.envRadixDeploymentBuilders[environment]; ok {
			for _, radixDeploymentBuilder := range radixDeploymentBuilders {
				_, err := radixClient.RadixV1().RadixDeployments(utils.GetEnvironmentNamespace(appName, environment)).
					Create(context.Background(), radixDeploymentBuilder.WithAppName(appName).BuildRD(), metav1.CreateOptions{})
				if err != nil {
					t.Fatal(err.Error())
				}
			}
		}
	}
	for _, pipelineJob := range scenario.pipelineJobs {
		_, _ = radixClient.RadixV1().RadixJobs(pipelineJob.GetNamespace()).Create(context.Background(), pipelineJob, metav1.CreateOptions{})
	}
	return radixClient
}

func setupLog(t *testing.T) {
	log.AddHook(test.NewTestLogHook(t, log.DebugLevel).
		ModifyFormatter(func(f *log.TextFormatter) { f.DisableTimestamp = true }))
}
