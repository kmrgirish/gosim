package simulation

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/jellevandenhooff/gosim/gosimruntime"
)

// Per-machine globals initialized in setupUserspace:

var (
	linuxOS          *LinuxOS
	gosimOS          *GosimOS // XXX: elsewhere? in machine itself?
	currentMachineID int
)

func CurrentMachineID() int {
	return currentMachineID
}

type gosimSlogHandler struct {
	inner slog.Handler
}

func (w gosimSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return w.inner.Enabled(ctx, level)
}

func (w gosimSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.Int("goroutine", gosimruntime.GetGoroutine()))
	r.AddAttrs(slog.Int("step", gosimruntime.Step()))
	return w.inner.Handle(ctx, r)
}

func (w gosimSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return gosimSlogHandler{
		inner: w.inner.WithAttrs(attrs),
	}
}

func (w gosimSlogHandler) WithGroup(name string) slog.Handler {
	return gosimSlogHandler{
		inner: w.inner.WithGroup(name),
	}
}

type gosimLogWriter struct{}

func (w gosimLogWriter) Write(b []byte) (n int, err error) {
	gosimruntime.WriteLog(b)
	return len(b), nil
}

func setupSlog(machineLabel string) {
	// We play a funny game with the logger. There exists a default slog.Logger in
	// every machine, since they all have their own set of globals. All loggers
	// point to the same LogOut writer and are set up using the configuration set
	// here.

	// TODO racedetector: make sure that using slog doesn't introduce sneaky
	// happens-before (this currently happens in the default handler which
	// uses both a pool and a lock)

	time.Local = time.UTC

	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("GOSIM_LOG_LEVEL"))); err != nil {
		panic(err)
	}

	// TODO: capture stdout? stderr?

	ho := slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}
	handler := slog.NewJSONHandler(gosimLogWriter{}, &ho)

	// set short file flag so that we'll capture source info. see slog.SetDefault internals
	// XXX: test this?
	log.SetFlags(log.Lshortfile)
	slog.SetDefault(slog.New(gosimSlogHandler{inner: handler}).With("machine", machineLabel))
}

func setupUserspace(gosimOS_ *GosimOS, linuxOS_ *LinuxOS, machineID int, label string) {
	// XXX: does logging work during global init?
	gosimruntime.InitGlobals(false, false)

	// yikes... how do we order this? should machine exist first? should these happen in an init() in here SOMEHOW? short-circuit this package?
	// XXX: provide directoyr
	gosimOS = gosimOS_
	linuxOS = linuxOS_
	currentMachineID = machineID

	setupSlog(label)
}
