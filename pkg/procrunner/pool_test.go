package procrunner

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProcRunnerPoolTestSuite struct {
	suite.Suite
}

func TestProcRunnerPoolTestSuite(t *testing.T) {
	suite.Run(t, new(ProcRunnerPoolTestSuite))
}

func (suite *ProcRunnerPoolTestSuite) SetupTest() {
}

func (suite *ProcRunnerPoolTestSuite) TestConcurrency() {
	pool := NewProcRunnerPool(2)
	suite.Equal(0, pool.Running())
	suite.Equal(2, pool.Max())
	runner1, err := pool.NewRunner("test.js", 32)
	suite.NoError(err)
	suite.Equal(1, pool.Running())
	suite.Equal(2, pool.Max())
	runner2, err := pool.NewRunner("test.js", 32)
	suite.NoError(err)
	suite.Equal(2, pool.Running())
	runner3, err := pool.NewRunner("test.js", 32)
	suite.Equal(err, ErrMaxReached)
	suite.Nil(runner3)
	runner1.Close()
	suite.Equal(1, pool.Running())
	runner4, err := pool.NewRunner("test.js", 32)
	suite.NoError(err)
	runner2.Close()
	runner4.Close()
	suite.Equal(0, pool.Running())
}
