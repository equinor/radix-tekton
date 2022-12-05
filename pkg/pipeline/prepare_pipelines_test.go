package pipeline

import (
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/stretchr/testify/assert"
	"testing"
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
