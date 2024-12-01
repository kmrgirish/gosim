package gosimlog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"runtime"
	"slices"
	"time"
)

type Stackframe struct {
	File     string `json:"file"`
	Function string `json:"function"`
	Line     int    `json:"line"`
}

type Log struct {
	Index int `json:"-"`

	Time        time.Time     `json:"time"`
	Level       slog.Level    `json:"level"`
	Msg         string        `json:"msg"`
	Source      *Stackframe   `json:"source"`
	Step        int           `json:"step"`
	RelatedStep int           `json:"relatedStep"`
	Stackframes []*Stackframe `json:"stackframes"`
	Machine     string        `json:"machine"`
	Goroutine   int           `json:"goroutine"`
	TraceKind   string        `json:"traceKind"`

	// map[string]any for extra fields
}

func ParseLog(logs []byte) []*Log {
	var out []*Log

	for _, line := range bytes.Split(logs, []byte("\n")) {
		var log Log
		if err := json.Unmarshal(line, &log); err != nil {
			// TODO: Is this ok?
			continue
		}
		out = append(out, &log)
	}

	return out
}

// Stack captures a stack with runtime.Callers and return a stackframes
// slog.Attr. If base is non-zero, Stack will skip frames until encountering
// base as a PC, which is helpful for calling Stack in a slog handler.
func Stack(skip int, base uintptr) slog.Attr {
	var stackRaw [256]uintptr
	n := runtime.Callers(skip+1, stackRaw[:])
	return StackFor(stackRaw[:n], base)
}

// StackFor returns a stackframes attr for a stack captured with runtime.Callers.
func StackFor(stack []uintptr, base uintptr) slog.Attr {
	var frames []Stackframe
	found := false
	if base == 0 {
		found = true
	} else {
		if idx := slices.Index(stack, base); idx != -1 {
			stack = stack[idx:]
			found = true
		}
	}
	if len(stack) > 0 {
		framesIter := runtime.CallersFrames(stack)
		for {
			frame, more := framesIter.Next()
			if !found && frame.PC == base {
				found = true
			}
			if found {
				frames = append(frames, Stackframe{
					Function: frame.Function,
					File:     frame.File,
					Line:     frame.Line,
				})
			}
			if !more {
				break
			}
		}
	}
	return slog.Any("stackframes", frames)
}
