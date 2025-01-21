package go123

import (
	"github.com/kmrgirish/gosim/gosimruntime"
	"github.com/kmrgirish/gosim/internal/simulation"
	"github.com/kmrgirish/gosim/internal/simulation/syscallabi"
	"github.com/kmrgirish/gosim/internal/testing"
)

func Runtime() gosimruntime.Runtime {
	return runtimeImpl{}
}

type runtimeImpl struct{}

var _ gosimruntime.Runtime = runtimeImpl{}

func (r runtimeImpl) Run(fn func()) {
	simulation.Runtime(fn)
}

func (r runtimeImpl) Setup() {
	syscallabi.Setup()
}

func (r runtimeImpl) TestEntrypoint(match string, skip string, tests []gosimruntime.Test) bool {
	return testing.Entrypoint(match, skip, tests)
}
