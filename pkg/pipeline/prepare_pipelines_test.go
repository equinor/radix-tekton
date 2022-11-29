package pipeline

import (
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ComponentHasChangedSource(t *testing.T) {
	var testScenarios = []struct {
		changedFolders []string
		sourceFolder   string
		expectedResult bool
	}{
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   ".",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "././",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "nonexistingdir/",
			expectedResult: false,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "nonexistingdir",
			expectedResult: false,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo/",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo/pages",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo/pages/",
			expectedResult: true,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo/pages/water",
			expectedResult: false,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "dynageo/pages/water/",
			expectedResult: false,
		},
		{
			changedFolders: []string{"dynageo/pages/tracers", "dynageo/pages/water_chemistry", "notebooks", "tests"},
			sourceFolder:   "notebooks",
			expectedResult: false,
		},
	}

	var applicationComponent v1.RadixComponent

	for _, testScenario := range testScenarios {
		applicationComponent =
			utils.AnApplicationComponent().
				WithName("client-component-1").
				WithSourceFolder(testScenario.sourceFolder).
				BuildComponent()
		sourceHasChanged := componentHasChangedSource("someEnv", &applicationComponent, testScenario.changedFolders)
		assert.Equal(t, sourceHasChanged, testScenario.expectedResult)
	}
}
