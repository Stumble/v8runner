package runner

import (
	"context"
	"fmt"
	"strings"

	v8 "github.com/stumble/v8go"
)

var (
	ErrorTimeout = fmt.Errorf("timeout")
)

type Option interface {
	Apply() error
}

type MaxHeapSizeOption struct {
	HeapSizeMB uint
}

func (o MaxHeapSizeOption) Apply() error {
	v8.SetFlags(fmt.Sprintf("--max-heap-size=%d", o.HeapSizeMB))
	return nil
}

// Runner is a JavaScript runner. It must be closed after use.
type Runner struct {
	fileName string
	vm       *v8.Isolate
	codeCtx  *v8.Context
	closed   bool
}

// NewRunner creates a new JavaScript runner.
func NewRunner(fileName string, options ...Option) (*Runner, error) {
	fileName = strings.TrimSuffix(fileName, ".js") + ".js"
	for _, opt := range options {
		err := opt.Apply()
		if err != nil {
			return nil, err
		}
	}
	vm := v8.NewIsolate()
	codeCtx := v8.NewContext(vm)
	return &Runner{
		fileName: fileName,
		vm:       vm,
		codeCtx:  codeCtx,
	}, nil
}

// Close free resources.
func (r *Runner) Close() {
	r.codeCtx.Close()
	r.vm.Dispose()
	r.closed = true
}

func (r *Runner) RunScript(ctx context.Context, script string) (*v8.Value, error) {
	if r.closed {
		return nil, fmt.Errorf("runner is closed")
	}
	val, err := r.runScript(ctx, script)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (r *Runner) CodeCtx() *v8.Context {
	return r.codeCtx
}

func (r *Runner) runScript(ctx context.Context, script string) (*v8.Value, error) {
	vals := make(chan *v8.Value, 1)
	errs := make(chan error, 1)
	go func() {
		val, err := r.codeCtx.RunScript(script, r.fileName)
		if err != nil {
			errs <- fmt.Errorf("failed to run script because: %w", err)
			return
		}
		vals <- val
	}()

	// Do not return error details in error.
	select {
	case val := <-vals:
		return val, nil
	case err := <-errs:
		return nil, err
	case <-ctx.Done():
		r.vm.TerminateExecution() // terminate the execution
		r.Close()                 // close the runner once the execution is terminated
		err := <-errs             // will get a termination error back from the running script
		return nil, fmt.Errorf("%w: %s", ErrorTimeout, err)
	}
}
