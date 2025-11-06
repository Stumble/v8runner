package main

import (
	"context"
	"fmt"

	"github.com/stumble/v8runner/pkg/procrunner"
)

func main() {
	const maxHeapSizeMB = 16
	const jsFileName = "expression.js"
	runner, err := procrunner.NewProcRunner(jsFileName, maxHeapSizeMB)
	if err != nil {
		panic(err)
	}
	defer runner.Close()

	res, err := runner.RunCodeJSON(
		context.Background(),
		"const f = (data) => { return {a: data.X, b: data.Y} };",
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
	res, err = runner.RunCodeJSON(context.Background(), "f({X: 1, Y: 2});")
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}
