package runner

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/stumble/v8runner/pkg/types"
)

type ReaderRunner struct {
	FileName      string
	MaxHeapSizeMB uint
	Input         io.Reader
	Output        io.Writer
}

// NewReaderRunner creates a new ReaderRunner that reads from input and writes to output.
func NewReaderRunner(input io.Reader, output io.Writer, fileName string, maxHeapSizeMB uint) (*ReaderRunner, error) {
	return &ReaderRunner{
		FileName:      fileName,
		MaxHeapSizeMB: maxHeapSizeMB,
		Input:         input,
		Output:        output,
	}, nil
}

// NewStdioRunner creates a new ReaderRunner that reads from stdin and writes to stdout.
func NewStdioRunner(fileName string, maxHeapSizeMB uint) (*ReaderRunner, error) {
	return NewReaderRunner(os.Stdin, os.Stdout, fileName, maxHeapSizeMB)
}

func (r *ReaderRunner) Process() error {
	ctx := context.Background()
	runner, err := NewRunner(r.FileName, MaxHeapSizeOption{HeapSizeMB: r.MaxHeapSizeMB})
	if err != nil {
		return fmt.Errorf("failed to create runner: %v", err)
	}
	defer runner.Close()

	in := gob.NewDecoder(r.Input)
	out := gob.NewEncoder(r.Output)

	for {
		var req types.RunCodeRequest
		err := in.Decode(&req)
		if err != nil {
			// end of input
			if err == io.EOF {
				return nil
			}
			// unexpected input, return error
			return fmt.Errorf("failed to decode req: %w", err)
		}

		val, err := runner.RunScript(ctx, req.Code)
		if err != nil {
			if err = out.Encode(errResult(req.ID, err)); err != nil {
				return err
			}
			continue
		}

		switch req.ResponseType {
		case types.RtnValueTypeNil:
			if err = out.Encode(nilResult(req.ID)); err != nil {
				return err
			}
			continue
		case types.RtnValueTypeJSON:
			if err = out.Encode(jsonResult(req.ID, runner.CodeCtx(), val)); err != nil {
				return err
			}
			continue
		default:
			if err = out.Encode(
				errResult(req.ID,
					fmt.Errorf("unknown response type: %s", req.ResponseType))); err != nil {
				return err
			}
		}
	}
}
