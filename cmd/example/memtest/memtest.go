package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof" //nolint: gosec

	"github.com/stumble/v8runner/pkg/procrunner"
)

func runApplyInt64() {
	runner, err := procrunner.NewProcRunner("expression.js", 16)
	if err != nil {
		panic(err)
	}
	defer runner.Close()
	ctx := context.Background()
	_, err = runner.RunCodeJSON(
		ctx,
		`let f = function(data){ return data.a + data.b === 579 ? 111 : 0; }`,
	)
	if err != nil {
		panic(err)
	}
	vJSON, err := runner.RunCodeJSON(ctx, `f({a: 123, b: 456})`)
	if err != nil {
		panic(err)
	}
	var rst int64
	err = json.Unmarshal([]byte(vJSON), &rst)
	if err != nil {
		panic(err)
	}
	if rst != 111 {
		panic("rst not 111")
	}
}

func main() {
	go func() {
		// http://localhost:6060/debug/pprof/
		log.Println(http.ListenAndServe("localhost:6060", nil)) //nolint: gosec
	}()
	for {
		runApplyInt64()
	}
}
