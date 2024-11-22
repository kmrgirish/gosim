//go:build sim

package behavior_test

import (
	"log"
	"log/slog"
	"os"
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

func TestStdoutStderr(t *testing.T) {
	os.Stdout.WriteString("hello")
	os.Stderr.WriteString("goodbye")
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
