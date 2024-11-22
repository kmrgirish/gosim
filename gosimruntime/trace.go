package gosimruntime

import (
	"context"
	"encoding/binary"
	"log/slog"
)

type tracer struct {
	step   int
	hash   fnv64
	logger *slog.Logger
}

func newTracer(logger *slog.Logger) *tracer {
	return &tracer{
		step:   0,
		hash:   newFnv64(),
		logger: logger,
	}
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=traceKey

type traceKey byte

const (
	traceKeyRunPick traceKey = iota
	tracekeyRunResult
	tracekeyLogWrite
	traceKeyCreating
	traceKeyRunStarted
	traceKeyRunFinished
	traceKeyTimeNow
)

func (t *tracer) recordIntInt(key traceKey, a, b uint64) {
	if inControlledNondeterminism() {
		panic("help")
	}

	t.hash.Hash([]byte{byte(key)})
	t.hash.HashInt(a)
	t.hash.HashInt(b)

	level := slog.LevelDebug
	if *forceTrace {
		level = slog.LevelError
	}
	if t.logger.Enabled(context.TODO(), level) {
		t.logger.LogAttrs(context.TODO(), level, "dettrace",
			slog.Int("step", t.step),
			slog.String("key", key.String()),
			slog.Int("a", int(a)),
			slog.Int("b", int(b)),
			slog.Int64("sum", int64(t.hash)))
	}
	t.step++
}

func (t *tracer) recordBytes(key traceKey, a []byte) {
	if inControlledNondeterminism() {
		panic("help")
	}

	t.hash.Hash([]byte{byte(key)})
	t.hash.Hash(a)

	level := slog.LevelDebug
	if *forceTrace {
		level = slog.LevelError
	}
	if t.logger.Enabled(context.TODO(), level) {
		t.logger.LogAttrs(context.TODO(), level, "dettrace",
			slog.Int("step", t.step),
			slog.String("key", key.String()),
			slog.String("a", string(a)),
			slog.Int64("sum", int64(t.hash)))
	}
	t.step++
}

func (t *tracer) finalize() []byte {
	var n [8]byte
	binary.LittleEndian.PutUint64(n[:], uint64(t.hash))
	return n[:]
}

type traceWriter struct {
	trace *tracer
}

func (t traceWriter) Write(p []byte) (n int, err error) {
	t.trace.recordBytes(tracekeyLogWrite, p)
	return len(p), nil
}
