//go:build sim

package behavior_test

import (
	"os"
	"testing"

	"github.com/jellevandenhooff/gosim"
)

func TestMachineHostname(t *testing.T) {
	name, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	expected := "main"
	if name != expected {
		t.Errorf("bad name %q, expected %q", name, expected)
	}

	m := gosim.NewMachine(gosim.MachineConfig{
		Label: "hello123",
		MainFunc: func() {
			name, err := os.Hostname()
			if err != nil {
				t.Fatal(err)
			}
			expected := "hello123"
			if name != expected {
				t.Errorf("bad name %q, expected %q", name, expected)
			}
		},
	})

	m.Wait()
}
