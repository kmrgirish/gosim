package examples_test

import (
	"testing"

	"github.com/jellevandenhooff/gosim"
)

func TestHello(t *testing.T) {
	t.Log(gosim.IsSim())
}
