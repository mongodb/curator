package operations

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/mongo-go-driver/bson"
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

	assert.True(t, names["tojson"])
	assert.True(t, names["fromjson"])
	assert.True(t, names["tobson"])
	assert.True(t, names["frombson"])
}

func TestBSONRoundtrip(t *testing.T) {
	ftdcFileName := "test_files/ftdc_example"
	bsonFileName := "test_files/bson_from_ftdc.bson"
	ftdcCopyFileName := "test_files/ftdc_from_bson"
	bsonRoundtripFileName := "test_files/bson_roundtrip"
	defer func() {
		os.Remove(bsonFileName)
		os.Remove(ftdcCopyFileName)
		os.Remove(bsonRoundtripFileName)
	}()

	cmd := exec.Command("../curator", "ftdc", "tobson", "--input", ftdcFileName, "--output", bsonFileName)
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "frombson", "--input", bsonFileName, "--output", ftdcCopyFileName)
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "tobson", "--input", ftdcCopyFileName, "--output", bsonRoundtripFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	f1, err := os.Open(bsonFileName)
	require.NoError(t, err)
	defer f1.Close()
	f2, err := os.Open(bsonRoundtripFileName)
	require.NoError(t, err)
	defer f2.Close()

	bsonDocOriginal := bson.NewDocument()
	for {
		_, err := bsonDocOriginal.ReadFrom(f1)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	bsonDocRoundtrip := bson.NewDocument()
	for {
		_, err := bsonDocRoundtrip.ReadFrom(f2)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	fmt.Println(bsonDocOriginal)
	assert.True(t, bsonDocOriginal.Equal(bsonDocRoundtrip))

	/*
		equal, err := compareFiles(bsonFileName, bsonRoundtripFileName)
		require.NoError(t, err)
		assert.True(t, equal)
	*/
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

	cmd := exec.Command("../curator", "ftdc", "tojson", "--input", ftdcFileName, "--output", jsonFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	cmd = exec.Command("../curator", "ftdc", "fromjson", "--input", jsonFileName, "--prefix", path.Join(tempDir, prefix))
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	output, err := ioutil.ReadDir(tempDir)
	require.NoError(t, err)
	require.Equal(t, len(output), 1)
	ftdcCopyFileName := path.Join(tempDir, output[0].Name())

	cmd = exec.Command("../curator", "ftdc", "tojson", "--input", ftdcCopyFileName, "--output", jsonRoundtripFileName)
	_, err = cmd.CombinedOutput()
	require.NoError(t, err)

	equal, err := compareFiles(jsonFileName, jsonRoundtripFileName)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestBSONOutput(t *testing.T) {
	fileName := "bsonTestFTDC"
	collector := ftdc.NewDynamicCollector(100)
	docs := make([]*bson.Document, 4)
	for i := 0; i < len(docs); i++ {
		docs[i] = randFlatDocument(4)
		collector.Add(docs[i])
	}
	output, err := collector.Resolve()
	require.NoError(t, err)
	require.NoError(t, ioutil.WriteFile(fileName, output, 0777))

	cmd := exec.Command("../curator", "ftdc", "tobson", "--input", fileName)
	output, err = cmd.CombinedOutput()
	require.NoError(t, err)

	docNum := 0
	reader := bytes.NewBuffer(output)
	for {
		bsonDoc := bson.NewDocument()
		_, err = bsonDoc.ReadFrom(reader)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.True(t, docNum < len(docs))
		assert.True(t, bsonDoc.Equal(docs[docNum]))
		docNum++
	}
}

func randFlatDocument(numKeys int) *bson.Document {
	doc := bson.NewDocument()
	for i := 0; i < numKeys; i++ {
		doc.Append(bson.EC.Int64(randStr(), rand.Int63n(int64(numKeys)*1)))
	}

	return doc
}

func randStr() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
