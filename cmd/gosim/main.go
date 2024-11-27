package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jellevandenhooff/gosim/internal/gosimtool"
	"github.com/jellevandenhooff/gosim/internal/translate"
)

const doc = `Gosim is a tool for working with gosim simulation testing framework.

Usage: gosim <command> [arguments]

The commands are:

    test           test packages
    debug          debug a test in a package
    build-tests    build packages for metatesting
    translate      translate packages
    help           print this help

The 'translate' command:

Usage: gosim translate [-race] [packages]

The translate command translates packages without running the resulting code.
The output will be placed in <module root>/.gosim/translated. Translation
is cached by package, so re-running translate after modifying some files
in the current package should be fast.

Packages should be listed as if they were arguments to 'go test' or 'go build'
command. All listed packages must be part of the current module. The current
module must have a dependency on gosim by importing
'github.com/jellevandenhooff/gosim' somewhere in the code.

Translate translates code with specific build flags sets. The GOOS is fixed to
linux, and the GOARCH is the one used to compile translate.
 
The -race flag translates all code with the race build tag set.

The 'test' command:

Usage: gosim test [-race] [-run=...] [-v] [packages]

The test command translates and runs tests for the specified packages.  It first
invokes translate, and then invokes 'go test' on the translated code, passing
through the -run and -v flags.

The 'debug' command:

Usage: gosim debug [-race] [-headless] -package=[package] -test=[test] -step=[step]

The debug command translates and runs a specific test using the delve debugger.
It first invokes translate, and then runs 'dlv test' on the specific test. The
-step flag is the step to pause at as seen in the logs from running 'gosim test'.

The -headless flag optionally runs delve in headless mode for use with an
external interface like an IDE.

The 'build-tests' command:

Usage: gosim build-tests [-race] [packages]

The build-tests command translates and then builds tests for use with the
metatesting package. Metatesting in a cached go test run requires pre-building
tests; see the metatesting package documentation for details.
`

func commandName(cmd string) string {
	return fmt.Sprintf("%s %s", path.Base(os.Args[0]), cmd)
}

func packageName(pkgPath string) string {
	return pkgPath[strings.LastIndex(pkgPath, "/")+1:]
}

func batchPackagesWithDifferentNames(packages []string) [][]string {
	var groups [][]string
	count := make(map[string]int)

	for _, pkg := range packages {
		lastName := packageName(pkg)
		group := count[lastName]
		count[lastName]++
		if group >= len(groups) {
			groups = append(groups, nil)
		}
		groups[group] = append(groups[group], pkg)
	}

	return groups
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func copyFile(from, to string, perm fs.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func copyFileIfChanged(src, dst string, perm fs.FileMode) error {
	// skip unchanged so we keep the old modified timestamp
	dstHash, err := hashFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if dstHash != nil {
		srcHash, err := hashFile(src)
		if err != nil {
			return err
		}
		if bytes.Equal(srcHash, dstHash) {
			// skip because unchanged, maybe log?
			return nil
		}
	}

	if err := copyFile(src, dst, perm); err != nil {
		return err
	}

	return nil
}

func writeFileIfDifferent(b []byte, dst string, perm fs.FileMode) error {
	// skip unchanged so we keep the old modified timestamp
	dstBytes, err := os.ReadFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if bytes.Equal(b, dstBytes) {
		// skip because unchanged, maybe log?
		return nil
	}

	if err := os.WriteFile(dst, b, perm); err != nil {
		return err
	}

	return nil
}

const toolsgoTemplate = `//go:build never

package main

import (
	_ "` + gosimtool.Module + `"
)
`

func main() {
	flag.Usage = func() {
		fmt.Printf(doc)
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := flag.Args()[0]
	cmdArgs := flag.Args()[1:]

	cfg := gosimtool.BuildConfig{
		GOOS:   "linux",
		GOARCH: runtime.GOARCH,
		Race:   false,
	}

	switch cmd {
	case "translate":
		translateflags := flag.NewFlagSet(commandName("translate"), flag.ExitOnError)
		race := translateflags.Bool("race", false, "build in -race mode")
		translateflags.Parse(cmdArgs)

		cfg.Race = *race

		_, err := translate.Translate(&translate.TranslateInput{
			Packages: translateflags.Args(),
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

	case "test":
		testflags := flag.NewFlagSet(commandName("test"), flag.ExitOnError)
		verbose := testflags.Bool("v", false, "verbose output")
		race := testflags.Bool("race", false, "build in -race mode")
		run := testflags.String("run", "", "tests to run (as in go test -run)")
		logformat := testflags.String("logformat", "pretty", "gosim log formatting: raw|indented|pretty")
		simtrace := testflags.String("simtrace", "", "set of a comma-separated traces to enable")
		testflags.Parse(cmdArgs)

		cfg.Race = *race

		packages := testflags.Args()
		if len(packages) == 0 {
			packages = []string{"."}
		}

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: packages,
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		name := "go"
		args := []string{"test"}

		// TODO: only for go1.23?
		args = append(args, "-ldflags=-checklinkname=0", "-tags=linkname")
		if *verbose {
			args = append(args, "-v")
		}
		if *race {
			args = append(args, "-race")
		}
		args = append(args, "-trimpath")
		if *run != "" {
			args = append(args, "-run", *run)
		}
		args = append(args, output.Packages...)
		if *logformat != "pretty" {
			// TODO: configure this default somewhere?
			args = append(args, "-logformat", *logformat)
		}
		if *simtrace != "" {
			args = append(args, "-simtrace", *simtrace)
		}

		cmd := exec.Command(name, args...)
		cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
		cmd.Dir = output.RootOutputDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			log.Fatal(err)
		}

	case "build-tests":
		// do we supply packages or does it just work?
		// for now, let's supply packages...

		modDir, err := gosimtool.FindGoModDir()
		if err != nil {
			log.Fatal(err)
		}

		testflags := flag.NewFlagSet(commandName("build-tests"), flag.ExitOnError)
		race := testflags.Bool("race", false, "build in -race mode")
		testflags.Parse(cmdArgs)

		cfg.Race = *race

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: testflags.Args(),
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		// tests are named after the last part of the directory, even if they
		// have a different internal name (nice.)

		groups := batchPackagesWithDifferentNames(output.Packages)

		// log.Println(groups)

		dstDir := filepath.Join(modDir, gosimtool.OutputDirectory, "metatest", cfg.AsDirname())
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			log.Fatal(err)
		}

		buildDir, err := os.MkdirTemp("", "gosimbuild")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(buildDir)

		for _, group := range groups {
			// copy over existing binaries to make it faster
			for _, pkg := range group {
				binName := packageName(pkg) + ".test"
				src := filepath.Join(dstDir, gosimtool.PreparedTestBinName(pkg))
				dst := filepath.Join(buildDir, binName)
				if err := copyFile(src, dst, 0o755); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						// ignore missing source files, this is just an optimization
						continue
					}
					log.Fatal(err)
				}
			}

			// TODO: only for go1.23?
			name := "go"
			args := []string{"test"}
			args = append(args, "-ldflags=-checklinkname=0", "-tags=linkname")
			if *race {
				args = append(args, "-race")
			}
			args = append(args, "-trimpath")

			args = append(args, "-c")
			args = append(args, "-o", buildDir)

			args = append(args, group...)

			cmd := exec.Command(name, args...)
			cmd.Dir = output.RootOutputDir

			// TODO: consider if we want to do this?
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				log.Fatal(err)
			}

			for _, pkg := range group {
				binName := packageName(pkg) + ".test"
				src := filepath.Join(buildDir, binName)
				dst := filepath.Join(dstDir, gosimtool.PreparedTestBinName(pkg))

				// skip unchanged so we keep the old modified timestamp
				if err := copyFileIfChanged(src, dst, 0o755); err != nil {
					log.Fatal(err)
				}

				depBytes, err := json.MarshalIndent(output.Deps[pkg], "", "  ") // log.Println(binName, pkg, output.Deps[pkg])
				if err != nil {
					log.Fatal(err)
				}

				if err := writeFileIfDifferent(depBytes, filepath.Join(dstDir, gosimtool.PreparedTestInfoName(pkg)), 0o644); err != nil {
					log.Fatal(err)
				}
			}
		}

	case "debug":
		// TODO: make -headless flag write launch configuration?
		// TODO: for -headless, make ctrl-c work?
		// TODO: for -headless, use delve api to send initial continue?

		debugflags := flag.NewFlagSet(commandName("debug"), flag.ExitOnError)
		race := debugflags.Bool("race", false, "build in -race mode")
		pkg := debugflags.String("package", "", "package path to debug")
		test := debugflags.String("test", "", "full test name to debug")
		step := debugflags.Int("step", 0, "step to break at")
		headless := debugflags.Bool("headless", false, "run headless for IDE debugging")
		debugflags.Parse(cmdArgs)

		modDir, err := gosimtool.FindGoModDir()
		if err != nil {
			log.Fatal(err)
		}

		cfg.Race = *race

		output, err := translate.Translate(&translate.TranslateInput{
			Packages: []string{*pkg},
			Cfg:      cfg,
		})
		if err != nil {
			log.Fatal(err)
		}

		if len(output.Packages) != 1 {
			log.Fatalf("expected 1 output packages, got %v", output.Packages)
		}
		translated := output.Packages[0]

		script := fmt.Sprintf(`continue
stepout`)
		scriptPath := filepath.Join(modDir, gosimtool.OutputDirectory, "debug-script")
		if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
			log.Fatal(err)
		}

		name := "dlv"
		flags := []string{
			"test",
			translated,
			"--build-flags=-ldflags=-checklinkname=0 -tags=linkname",
			// TODO: does this actually set linkname?
			// TODO: pass on -race
		}

		if *headless {
			flags = append(flags,
				"--listen=:2345",
				"--accept-multiclient",
				"--headless",
			)
			log.Printf("running delve in headless mode, connect with vscode using a launch configuration (.vscode/launch.json) like:\n" +
				`{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to dlv-dap server on localhost:2345",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}",
            "port": 2345,
            "host": "127.0.0.1",
            "debugAdapter": "dlv-dap",
            "stopOnEntry": true,
        }
    ]
}`)
		} else {
			flags = append(flags,
				"--init="+scriptPath,
			)
			log.Println("running delve in the terminal...")
		}

		flags = append(flags,
			"--",
			"-test.run",
			"^"+*test+"$",
			"-test.v",
			"-step-breakpoint="+fmt.Sprint(*step),
		)

		cmd := exec.Command(name, flags...)
		cmd.Dir = path.Join(modDir, gosimtool.OutputDirectory, "translated", cfg.AsDirname())
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		log.Println(cmd.Args)
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			log.Fatal(err)
		}

	case "prepare-selftest":
		prepareSelftest()

	default:
		flag.Usage()
		os.Exit(2)
	}
}

// TODO: test with and without gosim dependency? (script test?)
