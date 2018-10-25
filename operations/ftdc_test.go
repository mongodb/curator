package operations

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	defer f1.Close()

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

func TestBSON(t *testing.T) {
	// Test decompression and recompression to and from BSON.
	// This test assumes that decompression is reversible.
	ftdcFileName := "test_files/ftdc_example"
	bsonFileName := "test_files/bson_from_ftdc.bson"
	ftdcCopyFileName := "test_files/ftdc_copy"
	defer func() {
		os.Remove(bsonFileName)
		os.Remove(ftdcCopyFileName)
	}()

	cmd := exec.Command("../curator", "ftdc", "ftdctobson", "--ftdcPath", ftdcFileName, "--bsonPath", bsonFileName)
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "bsontoftdc", "--bsonPath", bsonFileName, "--ftdcPrefix", ftdcCopyFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	equal, err := compareFiles(ftdcFileName, ftdcCopyFileName)
	require.NoError(t, err)
	assert.True(t, equal)
}
