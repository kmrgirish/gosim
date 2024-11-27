//go:build sim

package behavior_test

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/jellevandenhooff/gosim"
)

func TestLogSLog(t *testing.T) {
	slog.Info("hello 10")
}

func TestLogMachineGoroutineTime(t *testing.T) {
	done := make(chan struct{})

	log.Printf("hi %s", "there")
	go func() {
		gosim.NewMachine(gosim.MachineConfig{
			Label: "inside",
			MainFunc: func() {
				time.Sleep(10 * time.Second)
				log.Print("hey")
				go func() {
					time.Sleep(10 * time.Second)
					log.Print("propagated")
					slog.Warn("foo", "bar", "baz", "counter", 20)
					close(done)
				}()
				<-done
			},
		})
	}()

	<-done
}

func TestLogStdoutStderr(t *testing.T) {
	os.Stdout.WriteString("hello\n")
	os.Stderr.WriteString("goodbye\n")
	log.Println("same goroutine log")
	slog.Info("same goroutine slog")
}

func TestLogForPrettyTest(t *testing.T) {
	if os.Getenv("TESTINGFAIL") != "1" {
		t.Skip()
	}

	slog.Info("hello info", "foo", "bar")
	start := time.Now()
	time.Sleep(20 * time.Second)

	m := gosim.NewSimpleMachine(func() {
		slog.Info("warn", "ok", "now", "delay", time.Since(start))
		log.Println("before")
		panic("help")
	})
	m.Wait()
	log.Println("never")
}

func init() {
	// logs during init will print to stdout/stderr bypassing the gosim handler
	if os.Getenv("LOGDURINGINIT") == "1" {
		fmt.Println("hello")
		slog.Info("help")
	}
}

func TestLogDuringInit(t *testing.T) {
	// logs in above init() should print once for main

	// logs should print for this machine also
	gosim.NewMachine(gosim.MachineConfig{
		Label:    "logm",
		MainFunc: func() {},
	}).Wait()
}

func TestLogTraceSyscall(t *testing.T) {
	// should print ENOSYS
	syscall.Syscall(9999, 0, 0, 0)

	// should print open, write, close
	if err := os.WriteFile("hello", []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
}
