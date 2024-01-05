package procrunner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ProcRunnerTestSuite is the test suite for ProcRunner.
// YOU MUST install the most recent v8runner binary before running this test.
// go install github.com/stumble/v8runner/cmd/v8runner
type ProcRunnerTestSuite struct {
	suite.Suite
}

func TestProcRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(ProcRunnerTestSuite))
}

func (suite *ProcRunnerTestSuite) SetupTest() {
}

func (suite *ProcRunnerTestSuite) TestBasic() {
	runner, err := NewProcRunner("expression.js", 16)
	suite.Require().NoError(err)
	res, err := runner.RunCodeJSON(context.Background(), "1+1")
	suite.NoError(err)
	suite.Equal("2", res)
	runner.Close()
}

func (suite *ProcRunnerTestSuite) TestTimeout() {
	runner, err := NewProcRunner("expression.js", 16)
	suite.Require().NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	res, err := runner.RunCodeJSON(ctx, "while(true){}")
	suite.Equal(ErrorTimeout, err)
	suite.Equal("", res)
	// timeout close the runner
	suite.True(runner.IsClosed())
	// safe to close twice
	runner.Close()
}

func (suite *ProcRunnerTestSuite) TestMemoryLimit() {
	runner, err := NewProcRunner("expression.js", 4)
	suite.Require().NoError(err)
	res, err := runner.RunCodeJSON(context.Background(), `
  let memoryHog = [];
  while (true) {
      memoryHog.push(new Array(1024 * 1024).fill('X')); // Allocate 1MB chunks of memory
  }
`)
	suite.Equal(ErrorKilled, err)
	suite.Equal("", res)
	// memory limit does not close the runner
	suite.False(runner.IsClosed())
	res2, err2 := runner.RunCodeJSON(context.Background(), "1+1")
	suite.NotNil(err2)
	suite.Equal("", res2)
}

func (suite *ProcRunnerTestSuite) TestNoReturnValue() {
	runner, err := NewProcRunner("expression.js", 16)
	suite.Require().NoError(err)
	res, err := runner.RunCodeJSON(context.Background(), "function f(){};")
	suite.NoError(err)
	suite.Equal("undefined", res)

	res, err = runner.RunCodeJSON(context.Background(), "null;")
	suite.NoError(err)
	suite.Equal("null", res)

	// safe to close twice
	runner.Close()
}

func (suite *ProcRunnerTestSuite) TestThrow() {
	runner, err := NewProcRunner("expression.js", 16)
	suite.Require().NoError(err)
	res, err := runner.RunCodeJSON(context.Background(), "throw 'this is a test error';")
	suite.Equal("failed to run script because: this is a test error", err.Error())
	suite.Equal("", res)
	// safe to close twice
	runner.Close()
}
