package gosimruntime

import (
	"encoding/json"
	"flag"
	"log"
	"maps"
	"os"
	"slices"
)

type Test struct {
	Name string
	Test any // func()
}

var (
	allTests      map[string]any
	allTestsSlice []Test
)

func SetAllTests(tests []Test) {
	allTests = make(map[string]any)
	allTestsSlice = tests
	for _, test := range tests {
		allTests[test.Name] = test.Test
	}
}

var metatest = flag.Bool("metatest", false, "")

// Copied in metatesting.RunConfig. Keep in sync.
type runConfig struct {
	Test     string
	Seed     int64
	ExtraEnv []string
	Simtrace string
}

// Copied in metatesting.RunResult. Keep in sync.
type runResult struct {
	Seed      int64
	Trace     []byte
	Failed    bool
	LogOutput []byte
	Err       string // TODO: reconsider this type?
}

type Runtime interface {
	Run(fn func())
	Setup()
	TestEntrypoint(match string, skip string, tests []Test) bool
}

var simtrace = flag.String("simtrace", "", "set of comma-separated traces to enable")

func TestMain(rt Runtime) {
	flag.Parse()

	rt.Setup()
	initializeRuntime(rt.Run)

	if !*metatest {
		// TODO: complain about unsupported test flags
		// parallel := flag.Lookup("test.parallel").Value.(flag.Getter).Get().(int)
		match := flag.Lookup("test.run").Value.(flag.Getter).Get().(string)
		skip := flag.Lookup("test.skip").Value.(flag.Getter).Get().(string)

		if err := parseTraceflagsConfig(*simtrace); err != nil {
			log.Fatal(err)
		}

		seed := int64(1)
		enableTracer := true
		captureLog := true
		logLevelOverride := "INFO"

		// TODO: allow -count, -seeds

		outerOk := true

		for _, test := range allTestsSlice {
			// log.Println("running", test.Name)
			result := run(func() {
				ok := rt.TestEntrypoint(match, skip, []Test{
					test,
				})
				if !ok {
					SetAbortError(ErrTestFailed)
				}
			}, seed, enableTracer, captureLog, logLevelOverride, makeConsoleLogger(os.Stderr), []string{})

			if result.Failed {
				outerOk = false
			}
		}

		if !outerOk {
			os.Exit(1)
		}
		os.Exit(0)
	}

	in := json.NewDecoder(os.Stdin)
	out := json.NewEncoder(os.Stdout)

	for {
		var req runConfig
		if err := in.Decode(&req); err != nil {
			log.Fatal(err)
		}

		// TODO: add "listtests" as a separate message type?
		if req.Test == "listtests" {
			if err := out.Encode(slices.Sorted(maps.Keys(allTests))); err != nil {
				log.Fatal(err)
			}
			continue
		}

		seed := req.Seed
		enableTracer := true
		captureLog := true
		logLevelOverride := "INFO"

		if err := parseTraceflagsConfig(req.Simtrace); err != nil {
			metaResult := runResult{
				Err: err.Error(),
			}
			if err := out.Encode(metaResult); err != nil {
				log.Fatal(err)
			}
			continue
		}

		result := run(func() {
			// TODO: add an entrypoint that takes a single test?
			// TODO: fail gracefully with non-existent tests?
			ok := rt.TestEntrypoint(req.Test, "", []Test{
				{
					Name: req.Test,
					Test: allTests[req.Test],
				},
			})
			if !ok {
				SetAbortError(ErrTestFailed)
			}
		}, seed, enableTracer, captureLog, logLevelOverride, makeConsoleLogger(os.Stderr), req.ExtraEnv)

		metaResult := runResult{
			Seed:      result.Seed,
			Trace:     result.Trace,
			Failed:    result.Failed,
			LogOutput: result.LogOutput,
		}
		if result.Err != nil {
			metaResult.Err = result.Err.Error()
		}

		if err := out.Encode(metaResult); err != nil {
			log.Fatal(err)
		}
	}
}
