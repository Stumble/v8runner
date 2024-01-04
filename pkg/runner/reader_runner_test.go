package runner

import (
	"bytes"
	"encoding/gob"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ReaderRunnerTestSuite struct {
	suite.Suite
}

func TestReaderRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ReaderRunnerTestSuite))
}

func (suite *ReaderRunnerTestSuite) SetupTest() {
}

func (suite *ReaderRunnerTestSuite) TestOneRequest() {
	for _, tc := range []struct {
		name string
		req  RunCodeRequest
		res  RunCodeResponse
	}{
		{
			name: "simpleInt",
			req: RunCodeRequest{
				ID:           "x",
				Code:         "1+1",
				ResponseType: RtnValueTypeJSON,
			},
			res: RunCodeResponse{
				ID:     "x",
				Error:  nil,
				Result: ptr("2"),
			},
		},
		{
			name: "simpleJSON",
			req: RunCodeRequest{
				ID:           "x",
				Code:         `let v = {a:1, b:"xx"}; v;`,
				ResponseType: RtnValueTypeJSON,
			},
			res: RunCodeResponse{
				ID:     "x",
				Error:  nil,
				Result: ptr(`{"a":1,"b":"xx"}`),
			},
		},
		{
			name: "nil",
			req: RunCodeRequest{
				ID:           "y",
				Code:         `const f = (a,b) => { return a + b; }`,
				ResponseType: RtnValueTypeNil,
			},
			res: RunCodeResponse{
				ID:     "y",
				Error:  nil,
				Result: nil,
			},
		},
		{
			name: "invalid code",
			req: RunCodeRequest{
				ID:           "e",
				Code:         `const f = }`,
				ResponseType: RtnValueTypeNil,
			},
			res: RunCodeResponse{
				ID:     "e",
				Error:  ptr("failed to run script because: SyntaxError: Unexpected token '}'"),
				Result: nil,
			},
		},
	} {
		buf := &bytes.Buffer{}
		writeToBuf := gob.NewEncoder(buf)
		err := writeToBuf.Encode(tc.req)
		suite.NoError(err)
		result := &strings.Builder{}
		runner, err := NewReaderRunner(buf, result, "test.js", 16)
		suite.NoError(err)
		suite.NotNil(runner)
		err = runner.Process()
		suite.Require().NoError(err)
		res := &RunCodeResponse{}
		readFromBuf := gob.NewDecoder(strings.NewReader(result.String()))
		err = readFromBuf.Decode(res)
		suite.NoError(err)
		suite.Equal(tc.res, *res)
	}
}

func (suite *ReaderRunnerTestSuite) Test2Requests() {
	buf := &bytes.Buffer{}
	writeToBuf := gob.NewEncoder(buf)
	err := writeToBuf.Encode(RunCodeRequest{
		ID:           "x",
		Code:         "let a = 1+1; a;",
		ResponseType: RtnValueTypeJSON,
	})
	suite.NoError(err)
	err = writeToBuf.Encode(RunCodeRequest{
		ID:           "y",
		Code:         "let b = a+4; b;",
		ResponseType: RtnValueTypeJSON,
	})
	suite.NoError(err)

	result := &strings.Builder{}
	runner, err := NewReaderRunner(buf, result, "test.js", 4)
	suite.NoError(err)
	suite.NotNil(runner)
	err = runner.Process()
	suite.Require().NoError(err)
	res := &RunCodeResponse{}
	readFromBuf := gob.NewDecoder(strings.NewReader(result.String()))
	err = readFromBuf.Decode(res)
	suite.NoError(err)
	suite.Equal(RunCodeResponse{
		ID:     "x",
		Error:  nil,
		Result: ptr("2"),
	}, *res)
	err = readFromBuf.Decode(res)
	suite.NoError(err)
	suite.Equal(RunCodeResponse{
		ID:     "y",
		Error:  nil,
		Result: ptr("6"),
	}, *res)
}

func (suite *ReaderRunnerTestSuite) TestPipeline() {
	// Create a pipe mimic from sender to receiver's stdin
	stdin, stdinWriter := io.Pipe()
	// Create a pipe mimic from receiver's stdout to reader
	stdoutReader, stdout := io.Pipe()

	runner, err := NewReaderRunner(stdin, stdout, "test.js", 4)
	suite.NoError(err)
	suite.NotNil(runner)

	var wg sync.WaitGroup

	// simulate the server (binary) process
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := runner.Process()
		if err != nil {
			panic(err)
		}
		suite.Require().NoError(err)
	}()

	// simulate the client process
	wg.Add(1)
	go func() {
		defer wg.Done()

		encoder := NewRunCodeRequestEncoder(stdinWriter)
		decoder := NewReadRunCodeResponseDecoder(stdoutReader)
		for _, tc := range []struct {
			req RunCodeRequest
			res RunCodeResponse
		}{
			{
				req: RunCodeRequest{
					ID:           "x",
					Code:         "const f = (a,b) => { return a + b; }",
					ResponseType: RtnValueTypeNil,
				},
				res: RunCodeResponse{
					ID:     "x",
					Error:  nil,
					Result: nil,
				},
			},
			{
				req: RunCodeRequest{
					ID:           "y",
					Code:         `let v = f(3,4); v;`,
					ResponseType: RtnValueTypeJSON,
				},
				res: RunCodeResponse{
					ID:     "y",
					Error:  nil,
					Result: ptr(`7`),
				},
			},
		} {
			err := encoder.Encode(tc.req)
			suite.NoError(err)
			res := RunCodeResponse{}
			err = decoder.Decode(&res)
			suite.NoError(err)
			suite.Equal(tc.res, res)
		}
		suite.NoError(stdinWriter.Close())
	}()

	wg.Wait()
}

func ptr[T any](s T) *T {
	return &s
}
