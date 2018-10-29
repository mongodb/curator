package operations

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func compareFiles(file1, file2 string) (bool, error) {
	chunkSize := 1024

	f1, err := os.Open(file1)
	if err != nil {
		return false, errors.Wrapf(err, "problem opening file '%s'", file1)
	}
	defer f1.Close()
	f2, err := os.Open(file2)
	if err != nil {
		return false, errors.Wrapf(err, "problem opening file '%s'", file2)
	}
	defer f2.Close()

	for {
		b1 := make([]byte, chunkSize)
		_, err1 := f1.Read(b1)

		b2 := make([]byte, chunkSize)
		_, err2 := f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil
			} else if err1 != io.EOF && err1 != nil {
				return false, errors.Wrapf(err2, "problem reading file '%s'", file1)
			} else if err2 != io.EOF && err1 != nil {
				return false, errors.Wrapf(err2, "problem reading file '%s'", file2)
			} else if err1 == io.EOF || err2 == io.EOF {
				return false, nil
			}
		}

		if !bytes.Equal(b1, b2) {
			return false, nil
		}

	}
}

func TestFTDCParentCommandHasExpectedProperties(t *testing.T) {
	cmd := FTDC()
	names := make(map[string]bool)

	for _, sub := range cmd.Subcommands {
		assert.IsType(t, cli.Command{}, sub)
		names[sub.Name] = true
	}

	assert.Len(t, cmd.Subcommands, 4)
	assert.Equal(t, cmd.Name, "ftdc")

	assert.True(t, names["ftdctojson"])
	assert.True(t, names["jsontoftdc"])
	assert.True(t, names["ftdctobson"])
	assert.True(t, names["bsontoftdc"])
}

func TestBSONRoundtrip(t *testing.T) {
	ftdcFileName := "test_files/ftdc_example"
	bsonFileName := "test_files/bson_from_ftdc.bson"
	ftdcCopyFileName := "test_files/ftdc_from_bson"
	bsonRoundtripFilename := "test_files/bson_roundtrip"
	defer func() {
		os.Remove(bsonFileName)
		os.Remove(ftdcCopyFileName)
		os.Remove(bsonRoundtripFilename)
	}()

	cmd := exec.Command("../curator", "ftdc", "ftdctobson", "--input", ftdcFileName, "--output", bsonFileName)
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "bsontoftdc", "--input", bsonFileName, "--output", ftdcCopyFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "ftdctobson", "--input", ftdcCopyFileName, "--output", bsonRoundtripFilename)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	equal, err := compareFiles(bsonFileName, bsonRoundtripFilename)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestJSONRoundtrip(t *testing.T) {
	ftdcFileName := "test_files/ftdc_example"
	jsonFileName := "test_files/json_from_ftdc.json"
	ftdcCopyDir := "ftdc_copies"
	prefix := "ftdc_from_json"
	jsonRoundtripFileName := "test_files/json_rountrip"
	tempDir, err := ioutil.TempDir("test_files", ftdcCopyDir)
	require.NoError(t, err)
	defer func() {
		os.Remove(jsonFileName)
		os.RemoveAll(tempDir)
		os.Remove(jsonRoundtripFileName)
	}()

	cmd := exec.Command("../curator", "ftdc", "ftdctojson", "--input", ftdcFileName, "--output", jsonFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "jsontoftdc", "--input", jsonFileName, "--prefix", path.Join(tempDir, prefix))
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	output, err := ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Equal(t, len(output), 1)
	ftdcCopyFileName := path.Join(tempDir, output[0].Name())

	cmd = exec.Command("../curator", "ftdc", "ftdctojson", "--input", ftdcCopyFileName, "--output", jsonRoundtripFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	equal, err := compareFiles(jsonFileName, jsonRoundtripFileName)
	require.NoError(t, err)
	assert.True(t, equal)
}
