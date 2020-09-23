package operations

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/evergreen-ci/birch"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestFTDCParentCommandHasExpectedProperties(t *testing.T) {
	cmd := FTDC()
	names := make(map[string]bool)

	for _, sub := range cmd.Subcommands {
		assert.IsType(t, cli.Command{}, sub)
		names[sub.Name] = true
	}

	assert.Len(t, cmd.Subcommands, 2)
	assert.Equal(t, cmd.Name, "ftdc")

	assert.True(t, names["import"])
	assert.True(t, names["export"])
}

func runFTDCCommand(cmd cli.Command, in, out string) error {
	flags := &flag.FlagSet{}
	_ = flags.String(input, in, "")
	_ = flags.String(output, out, "")
	ctx := cli.NewContext(nil, flags, nil)
	return cli.HandleAction(cmd.Action, ctx)
}

func TestBSONRoundtrip(t *testing.T) {
	tempDir, err := ioutil.TempDir(".", "test_dir")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	}()
	bsonOriginal := path.Join(tempDir, "original.bson")
	bsonRoundtrip := path.Join(tempDir, "roundtrip.bson")
	ftdcFromOriginal := path.Join(tempDir, "ftdc")
	err = createBSONFile(bsonOriginal, 3)
	require.NoError(t, err)

	require.NoError(t, runFTDCCommand(fromBSON(), bsonOriginal, ftdcFromOriginal))
	require.NoError(t, runFTDCCommand(toBSON(), ftdcFromOriginal, bsonRoundtrip))

	equal, err := compareFiles(bsonOriginal, bsonRoundtrip)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestCSVRoundtrip(t *testing.T) {
	tempDir, err := ioutil.TempDir(".", "test_dir")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	}()
	csvOriginal := path.Join(tempDir, "original.csv")
	csvRoundtrip := path.Join(tempDir, "roundtrip.csv")
	ftdcFromOriginal := path.Join(tempDir, "ftdc")
	err = createCSVFile(csvOriginal, 3)
	require.NoError(t, err)

	require.NoError(t, runFTDCCommand(fromCSV(), csvOriginal, ftdcFromOriginal))
	require.NoError(t, runFTDCCommand(toCSV(), ftdcFromOriginal, csvRoundtrip))

	equal, err := compareFiles(csvOriginal, csvRoundtrip)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestJSONRoundtrip(t *testing.T) {
	tempDir, err := ioutil.TempDir(".", "test_dir")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	}()
	jsonOriginal := path.Join(tempDir, "original.json")
	jsonRoundtrip := path.Join(tempDir, "roundtrip.json")
	ftdcFromOriginal := path.Join(tempDir, "ftdc")
	err = createJSONFile(jsonOriginal, 3)
	require.NoError(t, err)

	flags := &flag.FlagSet{}
	_ = flags.String(input, jsonOriginal, "")
	_ = flags.String(prefix, ftdcFromOriginal, "")
	_ = flags.Int(maxCount, 1, "")
	_ = flags.Duration(flush, time.Second, "")
	ctx := cli.NewContext(nil, flags, nil)
	require.NoError(t, cli.HandleAction(fromJSON().Action, ctx))
	require.NoError(t, runFTDCCommand(toJSON(), ftdcFromOriginal+".0", jsonRoundtrip))

	equal, err := compareFiles(jsonOriginal, jsonRoundtrip)
	require.NoError(t, err)
	assert.True(t, equal)
}

func randFlatDocument(numKeys int) *birch.Document {
	doc := birch.NewDocument()
	for i := 0; i < numKeys; i++ {
		doc.Append(birch.EC.Int64(randStr(), rand.Int63n(int64(numKeys)*1)))
	}

	return doc
}

func randStr() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func createBSONFile(name string, size int) error {
	file, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "failed to create new file")
	}
	defer func() {
		grip.Alert(file.Close())
	}()

	for i := 0; i < size; i++ {
		_, err := randFlatDocument(size).WriteTo(file)
		if err != nil {
			return errors.Wrap(err, "failed to write BSON file")
		}
	}
	return nil
}

func createCSVFile(name string, size int) error {
	file, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "failed to create new file")
	}
	defer func() { grip.Alert(file.Close()) }()

	csvw := csv.NewWriter(file)
	if err := csvw.Write([]string{"one", "two", "three", "four", "five", "six", "seven", "eight"}); err != nil {
		return errors.Wrap(err, "problem writing header row")
	}
	for i := 0; i < size; i++ {
		row := []string{
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
			strconv.Itoa(rand.Int()),
		}

		if err := csvw.Write(row); err != nil {
			return errors.WithStack(err)
		}
	}
	csvw.Flush()
	return nil
}

func createJSONFile(name string, size int) error {
	file, err := os.Create(name)
	if err != nil {
		return errors.Wrap(err, "failed to create new file")
	}
	defer func() {
		grip.Alert(file.Close())
	}()

	for i := 0; i < size; i++ {
		jsonMap := make(map[string]int64, size)
		for j := 0; j < size; j++ {
			jsonMap[randStr()] = rand.Int63n(int64(size))
		}
		jsonString, err := json.Marshal(jsonMap)
		if err != nil {
			return errors.Wrap(err, "failed to marshal json")
		}
		_, err = file.Write(append(jsonString, '\n'))
		if err != nil {
			return errors.Wrap(err, "failed to write json to file")
		}
	}
	return nil
}

func compareFiles(file1, file2 string) (bool, error) {
	chunkSize := 1024

	f1, err := os.Open(file1)
	if err != nil {
		return false, errors.Wrapf(err, "problem opening file '%s'", file1)
	}
	defer func() {
		grip.Alert(f1.Close())
	}()
	f2, err := os.Open(file2)
	if err != nil {
		return false, errors.Wrapf(err, "problem opening file '%s'", file2)
	}
	defer func() {
		grip.Alert(f2.Close())
	}()

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
