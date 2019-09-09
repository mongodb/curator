package jasperutil

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
)

func CreateProcs(ctx context.Context, opts *jasper.CreateOptions, manager jasper.Manager, num int) ([]jasper.Process, error) {
	catcher := grip.NewBasicCatcher()
	out := []jasper.Process{}
	for i := 0; i < num; i++ {
		optsCopy := *opts

		proc, err := manager.CreateProcess(ctx, &optsCopy)
		catcher.Add(err)
		if proc != nil {
			out = append(out, proc)
		}
	}

	return out, catcher.Resolve()
}
