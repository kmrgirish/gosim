package behavior_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/jellevandenhooff/gosim"
)

func setupRealDisk(t *testing.T) {
	if gosim.IsSim() {
		return
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Error(err)
		return
	}

	tmp, err := os.MkdirTemp("", t.Name())
	if err != nil {
		t.Error(err)
		return
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmp); err != nil {
			panic(err)
		}
	})

	if err := os.Chdir(tmp); err != nil {
		t.Error(err)
		return
	}

	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			panic(err)
		}
	})
}

func TestDiskBasic(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Sync(); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("hello")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskTruncateGrow(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Truncate(10); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("hello\x00\x00\x00\x00\x00")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskTruncateShrink(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Truncate(3); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("hel")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskOpenTruncate(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}
	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Write([]byte("goodbye")); err != nil {
		t.Error(err)
		return
	}
	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("goodbye")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskSyncDir(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile(".", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	// XXX: you canc sync a O_RDONLY dir?
	if err := f.Sync(); err != nil {
		t.Error(err)
		return
	}
}

func TestDiskAfterClose(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	checkClosedError := func(err error, op string) {
		t.Helper()
		pathErr, ok := err.(*os.PathError)
		if !ok {
			t.Error("not a path error")
			return
		}
		if pathErr.Op != op {
			t.Errorf("expected op %q, got %q", op, pathErr.Op)
		}
		if pathErr.Err != os.ErrClosed {
			t.Errorf("expected os.ErrClosed, got %v", pathErr.Err)
		}
		if pathErr.Path != "hello" {
			t.Errorf("expected \"hello\", got %q", pathErr.Path)
		}
	}

	err = f.Close()
	checkClosedError(err, "close")
	slog.Info("return value of close after close", "err", err.Error())

	n, err := f.Read(nil)
	checkClosedError(err, "read")
	slog.Info("return value of read after close", "n", n, "err", err.Error())

	n, err = f.Write([]byte("hello"))
	checkClosedError(err, "write")
	slog.Info("return value of write after close", "n", n, "err", err.Error())

	err = f.Sync()
	checkClosedError(err, "sync")
	slog.Info("return value of sync after close", "err", err.Error())
}

func TestDiskWriteAtEndOfFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 0); err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 3); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("helhello")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskWriteAtAfterEndOfFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 3); err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 3+5+3); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("\x00\x00\x00hello\x00\x00\x00hello")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskReadAtBasic(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 0); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	data := make([]byte, 5)
	n, err := f.ReadAt(data, 0)
	if err != nil {
		t.Error(err)
	}
	if n != 5 || !bytes.Equal(data, []byte("hello")) {
		t.Error("bad read", n, data)
	}

	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskReadAtEndOfFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 0); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	{
		data := make([]byte, 5)
		n, err := f.ReadAt(data, 3)
		if err != io.EOF {
			t.Error(err)
		}
		if n != 2 || !bytes.Equal(data, []byte("lo\x00\x00\x00")) {
			t.Error("bad read", n, data)
		}
	}

	{
		data := make([]byte, 5)
		n, err := f.ReadAt(data, 5)
		if err != io.EOF {
			t.Error(err)
		}
		if n != 0 || !bytes.Equal(data, []byte("\x00\x00\x00\x00\x00")) {
			t.Error("bad read", n, data)
		}
	}

	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskReadAtAfterEndOfFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("hello", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.WriteAt([]byte("hello"), 0); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("hello", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	data := make([]byte, 5)
	n, err := f.ReadAt(data, 10)
	if err != io.EOF {
		t.Error(err)
	}
	if n != 0 || !bytes.Equal(data, []byte("\x00\x00\x00\x00\x00")) {
		t.Error("bad read", n, data)
	}

	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskOpenMissingFile(t *testing.T) {
	setupRealDisk(t)

	_, err := os.OpenFile("foo", os.O_RDONLY, 0o644)
	if err == nil {
		t.Error(err)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("should be not exist")
	}
	slog.Info("error opening a non-existing file", "err", err.Error())
}

func TestDiskStatMissingFile(t *testing.T) {
	setupRealDisk(t)

	_, err := os.Stat("foo")
	if err == nil {
		t.Error(err)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("should be not exist")
	}
	slog.Info("error stat a non-existing file", "err", err.Error())
}

func TestDiskStatFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	fi, err := os.Stat("foo")
	if err != nil {
		t.Error(err)
		return
	}

	if fi.IsDir() {
		t.Error("should not be dir, not", fi.IsDir())
	}
	if fi.Name() != "foo" {
		t.Error("should be foo, not ", fi.Name())
	}
}

func TestDiskStatFd(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	fi, err := f.Stat()
	if err != nil {
		t.Error(err)
		return
	}

	if fi.IsDir() {
		t.Error("should not be dir, not", fi.IsDir())
	}
	if fi.Name() != "foo" {
		t.Error("should be foo, not ", fi.Name())
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	fi2, err := os.Stat("foo")
	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(fi, fi2) {
		t.Error("different")
	}
}

func TestDiskDeleteMissingFile(t *testing.T) {
	setupRealDisk(t)

	err := os.Remove("foo")
	if err == nil {
		t.Error(err)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("should be not exist", err)
	}
	slog.Info("error deleting a non-existing file", "err", err.Error())
}

func TestDiskDeleteFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	if _, err := os.Stat("foo"); err != nil {
		t.Error("missing file before delete", err)
		return
	}

	if err := os.Remove("foo"); err != nil {
		t.Error("remove failed", err)
		return
	}

	if _, err := os.Stat("foo"); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Error("delete did not work", err)
		return
	}
}

func TestDiskRenameFile(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	if _, err := os.Stat("foo"); err != nil {
		t.Error("missing file before rename", err)
		return
	}

	if err := os.Rename("foo", "bar"); err != nil {
		t.Error("rename failed", err)
		return
	}

	if _, err := os.Stat("foo"); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Error("file still there after rename", err)
		return
	}

	f, err = os.OpenFile("bar", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("hello")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

func TestDiskRenameMissingFile(t *testing.T) {
	setupRealDisk(t)

	err := os.Rename("foo", "bar")
	if err == nil {
		t.Error(err)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("should be not exist")
	}
	slog.Info("error renaming a non-existing file", "err", err.Error())
}

func TestDiskRenameFileReplace(t *testing.T) {
	setupRealDisk(t)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	f, err = os.OpenFile("bar", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := f.Write([]byte("old")); err != nil {
		t.Error(err)
		return
	}

	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	if _, err := os.Stat("foo"); err != nil {
		t.Error("missing file before rename", err)
		return
	}

	if _, err := os.Stat("bar"); err != nil {
		t.Error("missing file before rename", err)
		return
	}

	if err := os.Rename("foo", "bar"); err != nil {
		t.Error("rename failed", err)
		return
	}

	if _, err := os.Stat("foo"); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Error("file still there after rename", err)
		return
	}

	f, err = os.OpenFile("bar", os.O_RDONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("hello")) {
		t.Error("bad read", data)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}

type dirEntry struct {
	Name  string
	IsDir bool
}

func checkReadDir(t *testing.T, dir string, expected []dirEntry) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Error(err)
		return
	}

	var converted []dirEntry
	for _, entry := range entries {
		converted = append(converted, dirEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		})
	}

	if diff := cmp.Diff(expected, converted); diff != "" {
		t.Error(diff)
	}
}

func TestDiskReadDir(t *testing.T) {
	setupRealDisk(t)

	checkReadDir(t, ".", nil)

	f, err := os.OpenFile("foo", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}
	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	checkReadDir(t, ".", []dirEntry{
		{
			Name:  "foo",
			IsDir: false,
		},
	})

	f, err = os.OpenFile("bar", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Error(err)
		return
	}
	if err := f.Close(); err != nil {
		t.Error(err)
		return
	}

	checkReadDir(t, ".", []dirEntry{
		{
			Name:  "bar",
			IsDir: false,
		},
		{
			Name:  "foo",
			IsDir: false,
		},
	})

	if err := os.Rename("bar", "zz"); err != nil {
		t.Error(err)
		return
	}

	checkReadDir(t, ".", []dirEntry{
		{
			Name:  "foo",
			IsDir: false,
		},
		{
			Name:  "zz",
			IsDir: false,
		},
	})

	if err := os.Remove("foo"); err != nil {
		t.Error(err)
		return
	}

	checkReadDir(t, ".", []dirEntry{
		{
			Name:  "zz",
			IsDir: false,
		},
	})
}

// open dir
// rename dir
// delete open file
// stat dir?
// stat size?

// fix duplication of writes in tests?

// things to test:
// - error codes for...
//   closed, eof, wrong mode
// - concurrent reads/writes (same handle, different handle)
// - read/write at end of file (writeat with offset at file, after file end)
// - read ending at file (err = nil or eof or both?)
// - ???
