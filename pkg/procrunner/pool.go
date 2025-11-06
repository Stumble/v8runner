package procrunner

import (
	"fmt"
	"sync"
)

var ErrMaxReached = fmt.Errorf("max reached")

// ProcRunnerPool is a manager that manages a pool of ProcRunner.
// It can safely enforce the global memory limit by limiting the number of concurrent ProcRunners.
type ProcRunnerPool struct {
	running int
	max     int
	mu      sync.Mutex
}

func NewProcRunnerPool(maxConcurrent int) *ProcRunnerPool {
	return &ProcRunnerPool{
		running: 0,
		max:     maxConcurrent,
		mu:      sync.Mutex{},
	}
}

func (p *ProcRunnerPool) Running() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *ProcRunnerPool) Max() int {
	return p.max
}

func (p *ProcRunnerPool) NewRunner(filename string, maxheapsizemb uint) (*ProcRunner, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running >= p.max {
		return nil, ErrMaxReached
	}
	runner, err := NewProcRunner(filename, maxheapsizemb)
	if err != nil {
		return nil, err
	}
	runner.AddPostCloseFn(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.running--
	})
	p.running++
	return runner, nil
}
