package procrunner

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/stumble/v8runner/pkg/runner"
)

var (
	ErrorTimeout = fmt.Errorf("timeout")
	ErrorClosed  = fmt.Errorf("closed")
	ErrorKilled  = fmt.Errorf("killed")
)

// ProcRunner is a runner that spawn a new process to run v8 js.
// It can safely enforce the global memory limit and per-request timeout.
// ProcRunner is not supposed to be used concurrently, although it is safe to do so.
// ProcRunner must be closed after use.
type ProcRunner struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	encoder *gob.Encoder
	decoder *gob.Decoder

	mu     sync.Mutex
	seq    uint64
	closed bool
}

// NewProcRunner creates a new ProcRunner that runs the given file.
func NewProcRunner(fileName string, maxHeapSizeMB uint) (*ProcRunner, error) {
	// Create the command
	// Should be safe to pass these parameters because they are not user input.
	cmd := exec.Command("v8runner", "--file", fileName, "--max-heap", fmt.Sprintf("%d", maxHeapSizeMB)) //nolint:gosec

	// Set up the stdin, stdout, stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &ProcRunner{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		encoder: gob.NewEncoder(stdin),
		decoder: gob.NewDecoder(stdout),
	}, nil
}

func (r *ProcRunner) IsClosed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

func (r *ProcRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

func (r *ProcRunner) close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.cmd.Process.Kill()
}

// RunCodeJSON runs the given code and returns the JSON result.
// There are multiple possible outcomes:
//  1. The process is killed by the runner because of timeout.
//     In this case, RunCodeJSON will return ErrorTimeout, and the runner will be closed.
//  2. The process is killed by the runner because of memory limit.
//     In this case, RunCodeJSON will return ErrorKilled, but the runner will not be closed.
//     Subsequent calls to RunCodeJSON will return other Errors like broken pipe.
//  3. Successful execution.
//     a. If the process returns a valid JSON, RunCodeJSON will return the JSON.
//     b. If the process returns an error, RunCodeJSON will return the error.
func (r *ProcRunner) RunCodeJSON(ctx context.Context, code string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// don't run if closed
	if r.closed {
		return "", ErrorClosed
	}

	r.seq++

	vals := make(chan *string, 1)
	errs := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		req := runner.RunCodeRequest{
			ID:           fmt.Sprintf("%d", r.seq),
			Code:         code,
			ResponseType: runner.RtnValueTypeJSON,
		}
		err := r.encoder.Encode(req)
		if err != nil {
			errs <- err
			return
		}

		var res runner.RunCodeResponse
		err = r.decoder.Decode(&res)
		if err != nil {
			// error is EOF when the process is killed
			if err == io.EOF {
				errs <- ErrorKilled
				return
			}
			errs <- err
			return
		}
		if res.Error != nil {
			errs <- fmt.Errorf(*res.Error)
			return
		}
		if res.ID != req.ID {
			// should be impossible to reach here
			errs <- fmt.Errorf("unexpected id: %s", res.ID)
			return
		}
		vals <- res.Result
	}()

	select {
	case val := <-vals:
		return *val, nil
	case err := <-errs:
		return "", err
	case <-ctx.Done():
		_ = r.close()
		// prevent goroutine leak
		// Close() would have killed the process and should
		// send an EOF to the decoder.
		wg.Wait()
		return "", ErrorTimeout
	}
}
