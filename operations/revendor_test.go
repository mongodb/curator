package operations

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatedGlideFile(t *testing.T) {
	const (
		pkg    = "some-package"
		oldRev = "old-revision"
		rev    = "new-revision"
	)
	pkgLine := fmt.Sprintf("package: %s", pkg)
	oldRevLine := fmt.Sprintf("    version: %s", oldRev)
	revLine := fmt.Sprintf("    version: %s", rev)

	for testName, testCase := range map[string]func(t *testing.T, file *os.File){
		"FailsWithoutExistingFile": func(t *testing.T, file *os.File) {
			_, err := updatedGlideFile("invalid", pkg, rev)
			assert.Error(t, err)
		},
		"FailsWithoutExistingPackage": func(t *testing.T, file *os.File) {
			_, err := file.WriteString(oldRevLine)
			require.NoError(t, err)

			_, err = updatedGlideFile(file.Name(), pkg, rev)
			assert.Error(t, err)
		},
		"FailsWithoutExistingVersion": func(t *testing.T, file *os.File) {
			_, err := file.WriteString(pkgLine)
			require.NoError(t, err)

			_, err = updatedGlideFile(file.Name(), pkg, rev)
			assert.Error(t, err)
		},
		"FailsWithoutProperVersionLine": func(t *testing.T, file *os.File) {
			_, err := file.WriteString(strings.Join([]string{pkgLine, rev}, "\n"))
			require.NoError(t, err)

			_, err = updatedGlideFile(file.Name(), pkg, rev)
			assert.Error(t, err)
		},
		"SuccessfullyReplacesLine": func(t *testing.T, file *os.File) {
			_, err := file.WriteString(strings.Join([]string{pkgLine, oldRevLine}, "\n"))
			require.NoError(t, err)

			content, err := updatedGlideFile(file.Name(), pkg, rev)
			require.NoError(t, err)
			assert.Contains(t, content, pkgLine)
			assert.Contains(t, content, revLine)
			assert.NotContains(t, content, oldRevLine)
		},
	} {
		t.Run(testName, func(t *testing.T) {
			file, err := ioutil.TempFile("", "revendor")
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, file.Close())
				assert.NoError(t, os.Remove(file.Name()))
			}()
			testCase(t, file)
		})
	}
}
