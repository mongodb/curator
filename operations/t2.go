package operations

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

func getGennyMetadata(ctx context.Context, inputPath string) (ftdc.GennyOutputMetadata, error) {
	var gennyOutput ftdc.GennyOutputMetadata

	input, err := os.Open(inputPath)
	if err != nil {
		return gennyOutput, errors.Wrapf(err, "problem opening file '%s'", inputPath)
	}
	defer func() { grip.Warning((input.Close())) }()
	fileName := filepath.Base(inputPath)
	gennyOutput.Name = strings.Split(fileName, ".ftdc")[0]
	gennyOutput.Iter = ftdc.ReadChunks(ctx, input)

	// GetGennyTime sets gennyOutput.StartTime and gennyOutput.EndTime 
	// which exhausts the current chunk iterator and renders it
	// unusable for future tasks.
	gennyOutput = ftdc.GetGennyTime(ctx, gennyOutput)

	return gennyOutput, nil
}
