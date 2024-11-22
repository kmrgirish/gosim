//go:build !sim

package behavior_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/jellevandenhooff/gosim/metatesting"
)

func parseLog(t *testing.T, log []byte) []map[string]any {
	var logs []map[string]any
	for _, line := range bytes.Split(log, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("===")) || bytes.HasPrefix(line, []byte("---")) {
			// XXX: test this also, later?
			continue
		}
		if len(line) == 0 {
			logs = append(logs, nil)
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal(line, &msg); err != nil {
			t.Fatal(line, err)
		}
		delete(msg, "source")
		logs = append(logs, msg)
	}
	return logs
}

func TestMetaLogMachineGoroutineTime(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestLogMachineGoroutineTime",
		Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	actual := parseLog(t, run.LogOutput)
	expected := parseLog(t, []byte(`{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hi there","machine":"main","goroutine":4,"step":1}
{"time":"2020-01-15T14:10:13.000001234Z","level":"INFO","msg":"hey","machine":"inside","goroutine":6,"step":2}
{"time":"2020-01-15T14:10:23.000001234Z","level":"INFO","msg":"propagated","machine":"inside","goroutine":7,"step":3}
{"time":"2020-01-15T14:10:23.000001234Z","level":"WARN","msg":"foo","machine":"inside","bar":"baz","counter":20,"goroutine":7,"step":4}
`))

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("diff", diff)
	}
}

func TestMetaLogSLog(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestLogSLog",
		Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	actual := parseLog(t, run.LogOutput)
	expected := parseLog(t, []byte(`{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hello 10","machine":"main","goroutine":4,"step":1}
`))

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("diff", diff)
	}
}

func TestMetaStdoutStderr(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestStdoutStderr",
		Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	actual := parseLog(t, run.LogOutput)
	expected := parseLog(t, []byte(`{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hello","machine":"os","method":"stdout","from":"main","goroutine":1,"step":1}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"goodbye","machine":"os","method":"stderr","from":"main","goroutine":1,"step":2}
`))

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("diff", diff)
	}
}
