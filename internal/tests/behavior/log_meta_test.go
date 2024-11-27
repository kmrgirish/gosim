//go:build !sim

package behavior_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/jellevandenhooff/gosim/internal/race"
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

func TestLogMachineGoroutineTime(t *testing.T) {
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

func TestLogSLog(t *testing.T) {
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

func TestLogStdoutStderr(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestLogStdoutStderr",
		Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	actual := parseLog(t, run.LogOutput)
	expected := parseLog(t, []byte(`{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hello","machine":"main","method":"stdout","goroutine":4,"step":1}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"goodbye","machine":"main","method":"stderr","goroutine":4,"step":2}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"same goroutine log","machine":"main","goroutine":4,"step":3}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"same goroutine slog","machine":"main","goroutine":4,"step":4}
`))

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("diff", diff)
	}
}

func TestLogDuringInit(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestLogDuringInit",
		Seed: 1,
		ExtraEnv: []string{
			"LOGDURINGINIT=1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	actual := parseLog(t, run.LogOutput)
	expected := parseLog(t, []byte(`{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hello","machine":"main","method":"stdout","goroutine":3,"step":1}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"2020/01/15 14:10:03 INFO help","machine":"main","method":"stderr","goroutine":3,"step":2}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"hello","machine":"logm","method":"stdout","goroutine":5,"step":3}
{"time":"2020-01-15T14:10:03.000001234Z","level":"INFO","msg":"2020/01/15 14:10:03 INFO help","machine":"logm","method":"stderr","goroutine":5,"step":4}
`))

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error("diff", diff)
	}
}

func TestLogTraceSyscall(t *testing.T) {
	if race.Enabled {
		// TODO: repair, which will need a reasonable plan for printing data
		// buffers using syscallabi API.
		t.Skip("logging arguments causes race")
	}

	mt := metatesting.ForCurrentPackage(t)

	// no logs without simtrace
	run, err := mt.Run(t, &metatesting.RunConfig{
		Test: "TestLogTraceSyscall",
		Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(metatesting.SimplifyParsedLog(metatesting.ParseLog(run.LogOutput)), []string(nil)); diff != "" {
		t.Error("diff", diff)
	}

	// syscalls with simtrace
	run, err = mt.Run(t, &metatesting.RunConfig{
		Test:     "TestLogTraceSyscall",
		Seed:     1,
		Simtrace: "syscall",
	})
	if err != nil {
		t.Fatal(err)
	}

	// TODO: support snapshotting these logs?
	// TODO: include machine etc.?
	if diff := cmp.Diff(metatesting.SimplifyParsedLog(metatesting.ParseLog(run.LogOutput)), []string{
		"INFO unsupported syscall unknown (9999) 0 0 0 0 0 0",
		"INFO openat -100 hello 577 420",
		`INFO write 5 5 {"world"}`,
		"INFO close 5",
	}); diff != "" {
		t.Error("diff", diff)
	}
}
