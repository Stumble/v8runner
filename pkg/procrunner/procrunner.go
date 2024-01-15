package procrunner

import (
	// "bufio"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"

	"github.com/stumble/v8runner/pkg/types"
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

	mu  sync.Mutex
	seq uint64

	wg      sync.WaitGroup
	closeFn func()
	closed  atomic.Bool

	postCloseFn []func()
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

	proc := &ProcRunner{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		encoder: gob.NewEncoder(stdin),
		decoder: gob.NewDecoder(stdout),
		closeFn: sync.OnceFunc(func() {
			err := cmd.Process.Kill()
			if err != nil {
				log.Debug().Err(err).Msgf("v8 kill failed")
			}
		}),
	}

	proc.wg.Add(1)
	// uses Wait() to handle SIGCHLD to avoid zombie process.
	go func() {
		defer proc.wg.Done()
		_ = cmd.Wait()
		proc.closed.Store(true)
		// call postCloseFn only after the process is killed
		for _, f := range proc.postCloseFn {
			f()
		}
	}()
	// // handle stderr
	// proc.wg.Add(1)
	// go func() {
	// 	defer proc.wg.Done()
	// 	// Create a scanner to read stderr line by line
	// 	scanner := bufio.NewScanner(stderr)
	// 	for scanner.Scan() {
	// 		log.Debug().Msgf("v8 stderr: %s", scanner.Text())
	// 	}
	// 	// Check for errors in scanning
	// 	if err := scanner.Err(); err != nil {
	// 		log.Error().Err(err).Msg("error reading stderr")
	// 	}
	// }()
	return proc, nil
}

func (r *ProcRunner) IsClosed() bool {
	return r.closed.Load()
}

func (r *ProcRunner) Close() {
	r.closeFn()
	r.wg.Wait()
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
	if r.IsClosed() {
		return "", ErrorClosed
	}

	r.seq++

	vals := make(chan *string, 1)
	errs := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		req := types.RunCodeRequest{
			ID:           fmt.Sprintf("%d", r.seq),
			Code:         code,
			ResponseType: types.RtnValueTypeJSON,
		}
		err := r.encoder.Encode(req)
		if err != nil {
			errs <- err
			return
		}

		var res types.RunCodeResponse
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
		r.Close()
		// prevent goroutine leak
		// Close() would have killed the process and should
		// send an EOF to the decoder.
		wg.Wait()
		return "", ErrorTimeout
	}
}

// AddPostCloseFn adds a function to be called after the runner is closed.
// NOTE: NOT concurrency safe.
func (r *ProcRunner) AddPostCloseFn(f func()) {
	r.postCloseFn = append(r.postCloseFn, f)
}
