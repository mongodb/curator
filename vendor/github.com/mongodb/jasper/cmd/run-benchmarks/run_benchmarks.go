package main

import (
	"context"
	"fmt"

	"github.com/mongodb/jasper/benchmarks"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := benchmarks.RunLogging(ctx)
	if err != nil {
		fmt.Println(err)
	}
}
