package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/stumble/v8runner/pkg/procrunner"
)

// readFileToString reads the contents of the file specified by filename
// and returns it as a string.
func readFileToString(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run script.go <filename> <code>")
		os.Exit(1)
	}
	filename := os.Args[1]
	testCode := os.Args[2]

	const maxHeapSizeMB = 16
	runner, err := procrunner.NewProcRunner(filename, maxHeapSizeMB)
	if err != nil {
		panic(err)
	}
	defer runner.Close()

	code, err := readFileToString(filename)
	if err != nil {
		panic(err)
	}

	res, err := runner.RunCodeJSON(context.Background(), code)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)

	res, err = runner.RunCodeJSON(context.Background(), testCode)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}
