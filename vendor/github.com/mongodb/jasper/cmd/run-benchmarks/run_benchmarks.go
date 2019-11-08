package main

import (
	"context"
	"fmt"

	"github.com/mongodb/jasper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := jasper.RunLogBenchmarks(ctx)
	if err != nil {
		fmt.Println(err)
	}
}
