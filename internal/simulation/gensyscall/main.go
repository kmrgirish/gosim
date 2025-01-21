// Copyright 2018 The Go Authors. All rights reserved.  Use of this source code
// is governed by a BSD-style license that can be found at
// https://go.googlesource.com/go/+/refs/heads/master/LICENSE.

// This file is based on
// https://cs.opensource.google/go/x/sys/+/refs/tags/v0.26.0:unix/mksyscall.go

// This program generates wrappers for system calls in gosim to ergonomically
// invoke them, implement them, and hook them in the go standard library.
package main

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kmrgirish/gosim/internal/gosimtool"
	"github.com/kmrgirish/gosim/internal/translate"
)

// findAndReadSources looks up the paths of files in go packages path using `go
// list` and reads them.
func findAndReadSources(files []string) map[string][]byte {
	pkgs := make(map[string]string)
	for _, file := range files {
		pkg := path.Dir(file)
		pkgs[pkg] = ""
	}

	args := []string{"list", "-json=ImportPath,Dir"}
	for pkg := range pkgs {
		args = append(args, pkg)
	}
	cmd := exec.Command("go", args...)
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var output struct {
			ImportPath string
			Dir        string
		}
		if err := decoder.Decode(&output); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		pkgs[output.ImportPath] = output.Dir
	}

	contents := make(map[string][]byte)
	for _, file := range files {
		pkg := path.Dir(file)
		dir := pkgs[pkg]
		p := path.Join(dir, path.Base(file))
		bytes, err := os.ReadFile(p)
		if err != nil {
			log.Fatal(err)
		}
		contents[file] = bytes
	}

	return contents
}

type outputWithSortKey struct {
	sortKey string
	output  string
}

// outputSorter holds outputs to be sorted by key
type outputSorter struct {
	outputs []outputWithSortKey
}

func newOutputSorter() *outputSorter {
	return &outputSorter{}
}

func (o *outputSorter) append(key, value string) {
	o.outputs = append(o.outputs, outputWithSortKey{sortKey: strings.ToLower(key), output: value})
}

func (o *outputSorter) output() string {
	slices.SortStableFunc(o.outputs, func(a, b outputWithSortKey) int {
		return cmp.Compare(a.sortKey, b.sortKey)
	})
	var out strings.Builder
	for _, output := range o.outputs {
		out.WriteString(output.output)
	}
	return out.String()
}

func writeFormattedGoFile(path, contents string) {
	b := []byte("// Code generated by gensyscall. DO NOT EDIT.\n" + contents)
	b, err := format.Source(b)
	if err != nil {
		log.Printf("error formatting %s: %s; continuing so you can inspect...", path, err)
	}
	b = bytes.Replace(b, []byte("//\npackage"), []byte("package"), 1) // who knows why we need this but otherwise go fmt is mad
	if err := os.WriteFile(path, b, 0o644); err != nil {
		log.Fatal(err)
	}
}

var (
	regexpSys            = regexp.MustCompile(`^\/\/sys\t`)
	regexpSysNonblock    = regexp.MustCompile(`^\/\/sysnb\t`)
	regexpSysDeclaration = regexp.MustCompile(`^\/\/sys(nb)?\t(\w+)\(([^()]*)\)\s*(?:\(([^()]+)\))?\s*(?:=\s*((?i)_?SYS_[A-Z0-9_]+))?$`)
	regexpComma          = regexp.MustCompile(`\s*,\s*`)
	regexpSyscallName    = regexp.MustCompile(`([a-z])([A-Z])`)
	regexpParamKV        = regexp.MustCompile(`^(\S*) (\S*)$`)
)

type TypeKind string

const (
	TypePlain   TypeKind = ""
	TypePointer TypeKind = "*"
	TypeSlice   TypeKind = "[]"
)

// Param is function parameter
type Param struct {
	Orig string
	Name string
	Kind TypeKind
	Type string
}

// parseParam splits a parameter into name and type
func parseParam(p string) Param {
	ps := regexpParamKV.FindStringSubmatch(p)
	if ps == nil {
		fmt.Fprintf(os.Stderr, "malformed parameter: %s\n", p)
		os.Exit(1)
	}
	typ := ps[2]
	kind := TypePlain
	if strings.HasPrefix(typ, "[]") {
		typ = typ[2:]
		kind = TypeSlice
	}
	if strings.HasPrefix(typ, "*") {
		typ = typ[1:]
		kind = TypePointer
	}
	return Param{Orig: p, Name: ps[1], Kind: kind, Type: typ}
}

// parseParamList parses parameter list and returns a slice of parameters
func parseParamList(list string) []Param {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	var params []Param
	for _, paramStr := range regexpComma.Split(list, -1) {
		params = append(params, parseParam(paramStr))
	}
	return params
}

type syscallInfo struct {
	pkg                   string
	funcName              string
	sysName               string
	inputsStr, outputsStr string
	async                 bool
	synthetic             bool
}

var filterCommon = []string{
	"SYS_ACCEPT4",
	"SYS_BIND",
	"SYS_CHDIR",
	"SYS_CLOSE",
	"SYS_CONNECT",
	"SYS_FALLOCATE",
	"SYS_FCNTL",
	"SYS_FDATASYNC",
	"SYS_FLOCK",
	"SYS_FSTAT",
	"SYS_FSYNC",
	"SYS_FTRUNCATE",
	"SYS_GETCWD",
	"SYS_GETDENTS64",
	"SYS_GETPEERNAME",
	"SYS_GETPID",
	// "SYS_GETRANDOM",
	"SYS_GETSOCKNAME",
	"SYS_GETSOCKOPT",
	"SYS_LISTEN",
	"SYS_LSEEK",
	"SYS_MADVISE",
	"SYS_MKDIRAT",
	"SYS_MMAP",
	"SYS_MUNMAP",
	"SYS_OPENAT",
	"SYS_PREAD64",
	"SYS_PWRITE64",
	"SYS_READ",
	"SYS_RENAMEAT",
	"SYS_SETSOCKOPT",
	"SYS_SOCKET",
	"SYS_UNAME",
	"SYS_UNLINKAT",
	"SYS_WRITE",
}

var filterArm64 = append(slices.Clip(filterCommon), []string{
	"SYS_FSTATAT",
}...)

var filterAmd64 = append(slices.Clip(filterCommon), []string{
	"SYS_NEWFSTATAT",
}...)

var filter = map[string][]string{
	"arm64": filterArm64,
	"amd64": filterAmd64,
}

func parseSyscalls(paths []string, filter []string) []syscallInfo {
	sources := findAndReadSources(paths)
	filterMap := make(map[string]int)
	for _, name := range filter {
		filterMap[name] = 0
	}
	var parsedSyscalls []syscallInfo
	for file, contents := range sources {
		// fmt.Println(file) // , "\n", string(contents[:1024]))

		pkg := path.Dir(file)

		s := bufio.NewScanner(bytes.NewReader(contents))
		for s.Scan() {
			t := s.Text()
			if regexpSys.FindStringSubmatch(t) == nil && regexpSysNonblock.FindStringSubmatch(t) == nil {
				continue
			}
			// Line must be of the form
			//	func Open(path string, mode int, perm int) (fd int, errno error)
			// Split into name, in params, out params.
			f := regexpSysDeclaration.FindStringSubmatch(t)
			if f == nil {
				log.Fatalf("%s:%s\nmalformed //sys declaration\n", file, t)
				os.Exit(1)
			}

			funcName, inputsStr, outputsStr, sysName := f[2], f[3], f[4], f[5]

			if sysName == "" {
				sysName = "SYS_" + regexpSyscallName.ReplaceAllString(funcName, `${1}_$2`)
				sysName = strings.ToUpper(sysName)
			}

			if funcName == "readlen" || funcName == "writelen" {
				// help different signatures, skip
				continue
			}

			if _, ok := filterMap[sysName]; !ok {
				continue
			}
			filterMap[sysName]++

			parsedSyscalls = append(parsedSyscalls, syscallInfo{
				pkg:        pkg,
				funcName:   funcName,
				sysName:    sysName,
				inputsStr:  inputsStr,
				outputsStr: outputsStr,
			})
		}
	}
	for sysName, count := range filterMap {
		if count == 0 {
			log.Fatalf("help, did not see %s", sysName)
		}
	}

	// JANKY: Getrandom here for now because with the vDSO changes it no longer
	// appears as a comment.
	parsedSyscalls = append(parsedSyscalls, syscallInfo{
		pkg:        "golang.org/x/sys/unix",
		funcName:   "Getrandom",
		sysName:    "SYS_GETRANDOM",
		inputsStr:  "buf []byte, flags int",
		outputsStr: "n int, err error",
	})

	return parsedSyscalls
}

func dedupSyscalls(syscalls []syscallInfo) []syscallInfo {
	// TODO: dedup syscalls better???
	var deduped []syscallInfo
	seen := make(map[string]syscallInfo)
	for _, info := range syscalls {
		pkg, sysName, inputsStr, outputsStr := info.pkg, info.sysName, info.inputsStr, info.outputsStr

		if pkg == "golang.org/x/sys/unix" && sysName == "SYS_FSTATAT" {
			// slight difference w/ syscall, but ok
			continue
		}
		if pkg == "golang.org/x/sys/unix" && sysName == "SYS_NEWFSTATAT" {
			// slight difference w/ syscall, but ok
			continue
		}

		if previous, ok := seen[sysName]; ok {
			if inputsStr != previous.inputsStr || outputsStr != previous.outputsStr {
				log.Fatalf("help, %s has multiple different versions:\n%+v\n%+v", sysName, info, previous)
			}
			continue
		}
		seen[sysName] = info
		deduped = append(deduped, info)
	}
	return deduped
}

func ifaceName(sysName string) string {
	parts := strings.Split(sysName, "_")
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

var typemap = map[string]string{
	"RawSockaddrAny": "RawSockaddrAny",
	"Utsname":        "Utsname",
	"Stat_t":         "Stat_t",
	"_Socklen":       "Socklen",
}

type Usecase string

const (
	Proxy  Usecase = "Proxy"
	Switch Usecase = "Switch"
	Iface  Usecase = "Iface"
)

func toUintptr(val string, typ Param) string {
	if typ.Kind != TypePlain {
		panic("help")
	}
	if typ.Type == "bool" {
		return fmt.Sprintf("syscallabi.BoolToUintptr(%s)", val)
	} else if typ.Type == "error" {
		return fmt.Sprintf("syscallabi.ErrErrno(%s)", val)
	} else {
		return fmt.Sprintf("uintptr(%s)", val)
	}
}

func fromUintptr(val string, typ Param) string {
	if typ.Kind != TypePlain {
		panic("help")
	}
	if typ.Type == "bool" {
		return fmt.Sprintf("syscallabi.BoolFromUintptr(%s)", val)
	} else if typ.Type == "error" {
		return fmt.Sprintf("syscallabi.ErrnoErr(%s)", val)
	} else {
		return fmt.Sprintf("%s(%s)", maybeMapType(Switch, typ.Type), val)
	}
}

func maybeMapType(usecase Usecase, typ string) string {
	known, ok := typemap[typ]
	if !ok {
		return typ
	}
	if usecase == Proxy {
		return "simulation." + known
	}
	return known
}

func uppercase(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func writeProxies(rootDir string, pkgs []string, syscalls []syscallInfo, version string, arch string) {
	proxyTexts := make(map[string]*outputSorter)
	for _, pkg := range pkgs {
		proxyTexts[pkg] = newOutputSorter()
	}
	translateTexts := newOutputSorter()
	for _, info := range syscalls {
		pkg, funcName, sysName, inputsStr, outputsStr := info.pkg, info.funcName, info.sysName, info.inputsStr, info.outputsStr

		in := parseParamList(inputsStr)
		out := parseParamList(outputsStr)

		proxyName := translate.RewriteSelector(pkg, funcName)
		ifaceName := ifaceName(sysName)

		var proxyArgs, proxyCallArgs []string
		for _, p := range in {
			proxyArgs = append(proxyArgs, fmt.Sprintf("%s %s%s", p.Name, p.Kind, maybeMapType(Proxy, p.Type)))
			proxyCallArgs = append(proxyCallArgs, p.Name)
		}
		var proxyRets []string
		for _, p := range out {
			proxyRets = append(proxyRets, p.Orig)
		}
		var proxyRet string
		if len(out) > 0 {
			proxyRet = fmt.Sprintf(" (%s)", strings.Join(proxyRets, ", "))
		}
		proxyText := "" +
			fmt.Sprintf("func %s(%s)%s {\n", proxyName, strings.Join(proxyArgs, ", "), proxyRet) +
			fmt.Sprintf("\treturn simulation.Syscall%s(%s)\n", ifaceName, strings.Join(proxyCallArgs, ", ")) +
			"}\n\n"
		proxyTexts[pkg].append(funcName, proxyText)
		translateText := fmt.Sprintf("\t{Pkg: %s, Selector: %s}: {Pkg: hooks%sPackage},\n", strconv.Quote(pkg), strconv.Quote(funcName), uppercase(version))
		translateTexts.append(pkg+" "+funcName, translateText)
	}
	for _, pkg := range pkgs {
		file := strings.Replace(pkg, ".", "", -1)
		file = strings.Replace(file, "/", "_", -1)
		path := path.Join(rootDir, "internal/hooks/"+version, file+"_gen_"+arch+".go")

		writeFormattedGoFile(path, `//go:build linux
package `+version+`

import (
	"unsafe"

	"`+gosimtool.Module+`/internal/simulation"
)

// prevent unused imports
var _ unsafe.Pointer

`+proxyTexts[pkg].output())
	}
	writeFormattedGoFile(path.Join(rootDir, "internal/translate/hooks_"+version+"_"+arch+"_gen.go"), `package translate

func init() {
	hooksGensyscall`+uppercase(version)+`ByArch["`+arch+`"] = map[packageSelector]packageSelector{
`+translateTexts.output()+`}
}
`)
}

func writeSyscalls(outputPath string, syscalls []syscallInfo, sysnums map[string]int, isMachine bool) {
	osTypeName := "LinuxOS"
	osImplName := "linuxOS"
	osIfaceName := "linuxOSIface"
	if isMachine {
		osTypeName = "GosimOS"
		osImplName = "gosimOS"
		osIfaceName = "gosimOSIface"
	}

	callers := newOutputSorter()
	checks := newOutputSorter()
	ifaces := newOutputSorter()
	for _, info := range syscalls {
		sysName, inputsStr, outputsStr := info.sysName, info.inputsStr, info.outputsStr

		in := parseParamList(inputsStr)
		out := parseParamList(outputsStr)
		ifaceName := ifaceName(sysName)

		var callerArgs, dispatchArgs, ifaceArgs []string
		prepareCaller := ""
		prepareDispatch := ""
		cleanupCallerText := ""

		// XXX: thread through async

		for i, p := range in {
			if p.Kind == TypePointer {
				prepareCaller += fmt.Sprintf("\tsyscall.Ptr%d = %s\n", i, p.Name)
				cleanupCallerText += fmt.Sprintf("\tsyscall.Ptr%d = nil\n", i)
				prepareDispatch += fmt.Sprintf("\t\t%s := syscallabi.NewValueView(syscall.Ptr%d.(%s%s))\n", p.Name, i, p.Kind, maybeMapType(Switch, p.Type))
				ifaceArgs = append(ifaceArgs, fmt.Sprintf("%s syscallabi.ValueView[%s]", p.Name, maybeMapType(Iface, p.Type)))
			} else if p.Kind == TypePlain && (p.Type == "string" || p.Type == "unsafe.Pointer" || p.Type == "any") {
				prepareCaller += fmt.Sprintf("\tsyscall.Ptr%d = %s\n", i, p.Name)
				cleanupCallerText += fmt.Sprintf("\tsyscall.Ptr%d = nil\n", i)
				prepareDispatch += fmt.Sprintf("\t\t%s := syscall.Ptr%d.(%s)\n", p.Name, i, p.Type)
				ifaceArgs = append(ifaceArgs, fmt.Sprintf("%s %s", p.Name, p.Type))
			} else if p.Kind == TypeSlice {
				prepareCaller += fmt.Sprintf("\tsyscall.Ptr%d, syscall.Int%d = unsafe.SliceData(%s), uintptr(len(%s))\n", i, i, p.Name, p.Name)
				cleanupCallerText += fmt.Sprintf("\tsyscall.Ptr%d = nil\n", i)
				prepareDispatch += fmt.Sprintf("\t\t%s := syscallabi.NewSliceView(syscall.Ptr%d.(*%s), syscall.Int%d)\n", p.Name, i, maybeMapType(Switch, p.Type), i)
				ifaceArgs = append(ifaceArgs, fmt.Sprintf("%s syscallabi.SliceView[%s]", p.Name, maybeMapType(Iface, p.Type)))
			} else {
				prepareCaller += fmt.Sprintf("\tsyscall.Int%d = %s\n", i, toUintptr(p.Name, p))
				prepareDispatch += fmt.Sprintf("\t\t%s := %s\n", p.Name, fromUintptr(fmt.Sprintf("syscall.Int%d", i), p))
				ifaceArgs = append(ifaceArgs, fmt.Sprintf("%s %s", p.Name, maybeMapType(Iface, p.Type)))
			}

			callerArgs = append(callerArgs, fmt.Sprintf("%s %s%s", p.Name, p.Kind, maybeMapType(Iface, p.Type)))
			dispatchArgs = append(dispatchArgs, p.Name)
		}

		// if info.async {
		dispatchArgs = append(dispatchArgs, "syscall")
		ifaceArgs = append(ifaceArgs, "invocation *syscallabi.Syscall")
		// }

		parseCallerText := ""
		parseDispatchText := ""

		sysVal := fmt.Sprintf("unix.%s", sysName)

		var callerRets, dispatchRets, ifaceRets []string

		// Assign return values.
		for i, p := range out {
			callerRets = append(callerRets, p.Orig)
			dispatchRets = append(dispatchRets, p.Name)
			ifaceRets = append(ifaceRets, p.Orig)

			if p.Type == "string" {
				reg := fmt.Sprintf("syscall.RPtr%d", i)
				parseCallerText += fmt.Sprintf("\t%s = %s.(string)\n", p.Name, reg)
				parseDispatchText += fmt.Sprintf("\t\t%s = %s\n", reg, p.Name)
			} else {
				reg := fmt.Sprintf("syscall.R%d", i)
				if p.Name == "err" {
					reg = "syscall.Errno"
				}
				parseCallerText += fmt.Sprintf("\t%s = %s\n", p.Name, fromUintptr(reg, p))
				parseDispatchText += fmt.Sprintf("\t\t%s = %s\n", reg, toUintptr(p.Name, p))
			}
		}

		var callerRet, dispatchRet, ifaceRet string
		if len(out) > 0 {
			callerRet = fmt.Sprintf(" (%s)", strings.Join(callerRets, ", "))
			dispatchRet = fmt.Sprintf("%s := ", strings.Join(dispatchRets, ", "))
			ifaceRet = fmt.Sprintf(" (%s)", strings.Join(ifaceRets, ", "))
		}

		if info.async {
			dispatchRet = ""
			parseDispatchText = ""
		} else {
			parseDispatchText += "\t\tsyscall.Complete()\n"
		}

		if !info.synthetic {
			checkText := fmt.Sprintf("\tcase %s:\n", sysVal) +
				"\t\treturn true\n"
			checks.append(sysName, checkText)
		}

		ifaceText := fmt.Sprintf("\t%s(%s)%s\n", ifaceName, strings.Join(ifaceArgs, ", "), ifaceRet)
		ifaces.append(sysName, ifaceText)

		var invokeLog, retLog string
		if !isMachine {
			invokeLog = "\tif logSyscalls {\n" +
				"\t\tsyscall.Step = gosimruntime.Step()\n" +
				fmt.Sprintf("\t\tsyscallLogger.LogEntry%s(%s)\n", ifaceName, strings.Join(dispatchArgs, ", ")) +
				"\t}\n"
			retLog = "\tif logSyscalls {\n" +
				fmt.Sprintf("\t\tsyscallLogger.LogExit%s(%s)\n", ifaceName, strings.Join(append(dispatchArgs, dispatchRets...), ", ")) +
				"\t}\n"
		}

		callerText := "//go:norace\n" +
			fmt.Sprintf("func %s(%s)%s {\n", "Syscall"+ifaceName, strings.Join(callerArgs, ", "), callerRet) +
			"\tsyscall := syscallabi.GetGoroutineLocalSyscall()\n" +
			fmt.Sprintf("\tsyscall.Trampoline = trampoline%s\n", ifaceName) +
			fmt.Sprintf("\tsyscall.OS = %s\n", osImplName) +
			prepareCaller +
			invokeLog +
			fmt.Sprintf("\t%s.dispatchSyscall(syscall)\n", osImplName) +
			parseCallerText +
			retLog +
			cleanupCallerText +
			"\treturn\n" +
			"}\n\n"
		callers.append(sysName, callerText)

		dispatchText := "//go:norace\n" +
			fmt.Sprintf("func trampoline%s(syscall *syscallabi.Syscall) {\n", ifaceName) +
			prepareDispatch +
			fmt.Sprintf("\t\t%ssyscall.OS.(*%s).%s(%s)\n", dispatchRet, osTypeName, ifaceName, strings.Join(dispatchArgs, ", ")) +
			parseDispatchText +
			"}\n\n"
		callers.append(sysName, dispatchText)

		if !isMachine {
			loggerText := fmt.Sprintf("func (baseSyscallLogger) LogEntry%s(%s) {\n", ifaceName, strings.Join(append(callerArgs, "syscall *syscallabi.Syscall"), ", ")) +
				fmt.Sprintf("\tlogSyscallEntry(%s, syscall)\n", strconv.Quote(ifaceName)) +
				"}\n\n" +
				fmt.Sprintf("func (baseSyscallLogger) LogExit%s(%s) {\n", ifaceName, strings.Join(append(append(callerArgs, "syscall *syscallabi.Syscall"), callerRets...), ", ")) +
				fmt.Sprintf("\tlogSyscallExit(%s, syscall)\n", strconv.Quote(ifaceName)) +
				"}\n\n"
			callers.append(sysName, loggerText)
		}
	}

	output := ""
	if !isMachine {
		output += "//go:build linux\n"
	}
	output += fmt.Sprintf(`package simulation

import (
	"unsafe"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"`+gosimtool.Module+`/gosimruntime"
	"`+gosimtool.Module+`/internal/simulation/fs"
	"`+gosimtool.Module+`/internal/simulation/syscallabi"
)

// prevent unused imports
var (
	_ unsafe.Pointer
	_ time.Duration
	_ syscallabi.ValueView[any]
	_ syscall.Errno
	_ fs.InodeInfo
	_ unix.Errno
	_ = gosimruntime.GOOS
)

// %s is the interface *%s must implement to work
// with HandleSyscall. The interface is not used but helpful for implementing
// new syscalls.
type %s interface {
`, osIfaceName, osTypeName, osIfaceName) + ifaces.output() + fmt.Sprintf(`}

var _ %s = &%s{}

//go:norace
func (os *%s) dispatchSyscall(s *syscallabi.Syscall) {
	// XXX: help this happens for globals in os
	if os == nil {
		s.Errno = uintptr(syscall.ENOSYS)
		return
	}
	os.dispatcher.Dispatch(s)
}

`, osIfaceName, osTypeName, osTypeName) + callers.output()

	if !isMachine {
		output += `

type baseSyscallLogger struct {}
type customSyscallLogger struct { baseSyscallLogger }
var syscallLogger customSyscallLogger

func IsHandledSyscall(trap uintptr) bool {
	switch (trap) {
` + checks.output() + `	default:
		return false
	}
}
` + formatSysnums(sysnums)
	}

	writeFormattedGoFile(outputPath, output)
}

var machineCalls = []syscallInfo{
	{
		sysName:    "SET_SIMULATION_TIMEOUT",
		inputsStr:  "timeout time.Duration",
		outputsStr: "err error",
		synthetic:  true,
	},
	{
		sysName:    "SET_CONNECTED",
		inputsStr:  "a string, b string, connected bool",
		outputsStr: "err error",
		synthetic:  true,
	},
	{
		sysName:    "SET_DELAY",
		inputsStr:  "a string, b string, delay time.Duration",
		outputsStr: "err error",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_NEW",
		inputsStr:  "label string, addr string, program any",
		outputsStr: "machineID int",
		synthetic:  true,
	},
	{
		sysName:   "MACHINE_STOP",
		inputsStr: "machineID int, graceful bool",
		synthetic: true,
	},
	{
		sysName:   "MACHINE_INODE_INFO",
		inputsStr: "machineID int, inodeID int, info *fs.InodeInfo",
		synthetic: true,
	},
	{
		sysName:   "MACHINE_WAIT",
		inputsStr: "machineID int",
		async:     true,
		synthetic: true,
	},
	{
		sysName:    "MACHINE_RECOVER_INIT",
		inputsStr:  "machineID int, program any",
		outputsStr: "iterID int",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_RECOVER_NEXT",
		inputsStr:  "iterID int",
		outputsStr: "machineID int, ok bool",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_RECOVER_RELEASE",
		inputsStr:  "iterID int",
		outputsStr: "",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_RESTART",
		inputsStr:  "machineID int, partialDisk bool",
		outputsStr: "err error",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_GET_LABEL",
		inputsStr:  "machineID int",
		outputsStr: "label string, err error",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_SET_BOOT_PROGRAM",
		inputsStr:  "machineID int, program any",
		outputsStr: "err error",
		synthetic:  true,
	},
	{
		sysName:    "MACHINE_SET_SOMETIMES_CRASH_ON_SYNC",
		inputsStr:  "machineID int, crash bool",
		outputsStr: "err error",
		synthetic:  true,
	},
}

var (
	regexpSysMatch  = regexp.MustCompile(`SYS_`)
	regexpSysNumber = regexp.MustCompile(`^\s+(SYS_[A-Z0-9_]*)\s+=\s+([0-9]+)$`)
)

func parseSysnums(paths []string) map[string]int {
	sources := findAndReadSources(paths)

	result := make(map[string]int)

	for _, contents := range sources {
		// fmt.Println(file) // , "\n", string(contents[:1024]))

		s := bufio.NewScanner(bytes.NewReader(contents))
		for s.Scan() {
			t := s.Text()

			if regexpSysMatch.FindStringSubmatch(t) == nil {
				continue
			}

			f := regexpSysNumber.FindStringSubmatch(t)
			if f == nil {
				log.Fatalf("failed to match line %s", t)
			}

			name, number := f[1], f[2]

			parsedNumber, err := strconv.Atoi(number)
			if err != nil {
				log.Fatalf("parsing line %s: %s", t, err)
			}

			result[name] = parsedNumber
		}
	}

	return result
}

func formatSysnums(sysnums map[string]int) string {
	sorted := newOutputSorter()
	for name, num := range sysnums {
		sorted.append(fmt.Sprintf("%08d", num), fmt.Sprintf("\tcase unix.%s:// %d\n return %s\n", name, num, strconv.Quote(name)))
	}

	return fmt.Sprintf(`

func SyscallName(trap uintptr) string {
	switch trap {
		%s
	default:
		return "unknown"
	}
}
`, sorted.output())
}

func main() {
	rootDir, err := gosimtool.FindGoModDir()
	if err != nil {
		log.Fatal(err)
	}

	for _, arch := range []string{"arm64", "amd64"} {
		pkgs := []string{
			"syscall",
			"golang.org/x/sys/unix",
		}

		syscalls := parseSyscalls([]string{
			"syscall/syscall_linux.go",
			"syscall/syscall_linux_" + arch + ".go",
			"golang.org/x/sys/unix/syscall_linux.go",
			"golang.org/x/sys/unix/syscall_linux_" + arch + ".go",
		}, filter[arch])

		sysnums := parseSysnums([]string{
			"golang.org/x/sys/unix/zsysnum_linux_" + arch + ".go",
		})

		writeProxies(rootDir, pkgs, syscalls, "go123", arch)
		deduped := dedupSyscalls(syscalls)

		deduped = append(deduped, []syscallInfo{
			{
				sysName:    "POLL_OPEN",
				inputsStr:  "fd int, desc *syscallabi.PollDesc",
				outputsStr: "code int",
				synthetic:  true,
			},
			{
				sysName:    "POLL_CLOSE",
				inputsStr:  "fd int, desc *syscallabi.PollDesc",
				outputsStr: "code int",
				synthetic:  true,
			},
		}...)

		writeSyscalls(path.Join(rootDir, "internal/simulation/linux_gen_"+arch+".go"), deduped, sysnums, false)
	}

	writeSyscalls(path.Join(rootDir, "internal/simulation/gosim_gen.go"), machineCalls, nil, true)
}
