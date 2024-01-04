package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/stumble/v8runner/internal/info"
	"github.com/stumble/v8runner/pkg/runner"
)

var (
	fileName = flag.String("file", "runner.js", "file to run")
	maxHeap  = flag.Uint("max-heap", 16, "max heap size in MB")
)

func main() {
	fmt.Fprintf(os.Stderr, "v8runner version: %s\n", info.GetVersion())
	r, err := runner.NewStdioRunner(*fileName, *maxHeap)
	if err != nil {
		log.Fatalf("failed to create runner: %s", err)
	}
	err = r.Process()
	if err != nil {
		log.Fatalf("failed to process because: %s", err)
	}
}
