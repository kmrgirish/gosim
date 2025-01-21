//go:build !sim

package behavior_test

import (
	"testing"

	"github.com/kmrgirish/gosim/metatesting"
)

func TestMetaDeterministic(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	metatesting.CheckDeterministic(t, mt)
}

func TestMetaSeeds(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	metatesting.CheckSeeds(t, mt, 5)
}

func TestGosim(t *testing.T) {
	runner := metatesting.ForCurrentPackage(t)
	runner.RunAllTests(t)
}
