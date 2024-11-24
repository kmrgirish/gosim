//go:build linux

// Code generated by gensyscall. DO NOT EDIT.
package simulation

import (
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/jellevandenhooff/gosim/internal/simulation/fs"
	"github.com/jellevandenhooff/gosim/internal/simulation/syscallabi"
)

// prevent unused imports
var (
	_ unsafe.Pointer
	_ time.Duration
	_ syscallabi.ValueView[any]
	_ syscall.Errno
	_ fs.InodeInfo
	_ unix.Errno
)

// linuxOSIface is the interface *LinuxOS must implement to work
// with HandleSyscall. The interface is not used but helpful for implementing
// new syscalls.
type linuxOSIface interface {
	PollClose(fd int, desc syscallabi.ValueView[syscallabi.PollDesc]) (code int)
	PollOpen(fd int, desc syscallabi.ValueView[syscallabi.PollDesc]) (code int)
	SysAccept4(s int, rsa syscallabi.ValueView[RawSockaddrAny], addrlen syscallabi.ValueView[Socklen], flags int) (fd int, err error)
	SysBind(s int, addr unsafe.Pointer, addrlen Socklen) (err error)
	SysChdir(path string) (err error)
	SysClose(fd int) (err error)
	SysConnect(s int, addr unsafe.Pointer, addrlen Socklen) (err error)
	SysFallocate(fd int, mode uint32, off int64, len int64) (err error)
	SysFcntl(fd int, cmd int, arg int) (val int, err error)
	SysFdatasync(fd int) (err error)
	SysFlock(fd int, how int) (err error)
	SysFstat(fd int, stat syscallabi.ValueView[Stat_t]) (err error)
	SysFstatat(dirfd int, path string, stat syscallabi.ValueView[Stat_t], flags int) (err error)
	SysFsync(fd int) (err error)
	SysFtruncate(fd int, length int64) (err error)
	SysGetcwd(buf syscallabi.SliceView[byte]) (n int, err error)
	SysGetdents64(fd int, buf syscallabi.SliceView[byte]) (n int, err error)
	SysGetpeername(fd int, rsa syscallabi.ValueView[RawSockaddrAny], addrlen syscallabi.ValueView[Socklen]) (err error)
	SysGetpid() (pid int)
	SysGetrandom(buf syscallabi.SliceView[byte], flags int) (n int, err error)
	SysGetsockname(fd int, rsa syscallabi.ValueView[RawSockaddrAny], addrlen syscallabi.ValueView[Socklen]) (err error)
	SysGetsockopt(s int, level int, name int, val unsafe.Pointer, vallen syscallabi.ValueView[Socklen]) (err error)
	SysListen(s int, n int) (err error)
	SysLseek(fd int, offset int64, whence int) (off int64, err error)
	SysMadvise(b syscallabi.SliceView[byte], advice int) (err error)
	SysMkdirat(dirfd int, path string, mode uint32) (err error)
	SysMmap(addr uintptr, length uintptr, prot int, flags int, fd int, offset int64) (xaddr uintptr, err error)
	SysMunmap(addr uintptr, length uintptr) (err error)
	SysOpenat(dirfd int, path string, flags int, mode uint32) (fd int, err error)
	SysPread64(fd int, p syscallabi.SliceView[byte], offset int64) (n int, err error)
	SysPwrite64(fd int, p syscallabi.SliceView[byte], offset int64) (n int, err error)
	SysRead(fd int, p syscallabi.SliceView[byte]) (n int, err error)
	SysRenameat(olddirfd int, oldpath string, newdirfd int, newpath string) (err error)
	SysSetsockopt(s int, level int, name int, val unsafe.Pointer, vallen uintptr) (err error)
	SysSocket(domain int, typ int, proto int) (fd int, err error)
	SysUname(buf syscallabi.ValueView[Utsname]) (err error)
	SysUnlinkat(dirfd int, path string, flags int) (err error)
	SysWrite(fd int, p syscallabi.SliceView[byte]) (n int, err error)
}

var _ linuxOSIface = &LinuxOS{}

//go:norace
func (os *LinuxOS) dispatchSyscall(s *syscallabi.Syscall) {
	// XXX: help this happens for globals in os
	if os == nil {
		s.Errno = uintptr(syscall.ENOSYS)
		return
	}
	os.dispatcher.Dispatch(s)
}

//go:norace
func SyscallPollClose(fd int, desc *syscallabi.PollDesc) (code int) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolinePollClose
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1 = desc
	linuxOS.dispatchSyscall(syscall)
	code = int(syscall.R0)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolinePollClose(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	desc := syscallabi.NewValueView(syscall.Ptr1.(*syscallabi.PollDesc))
	code := syscall.OS.(*LinuxOS).PollClose(fd, desc)
	syscall.R0 = uintptr(code)
	syscall.Complete()
}

//go:norace
func SyscallPollOpen(fd int, desc *syscallabi.PollDesc) (code int) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolinePollOpen
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1 = desc
	linuxOS.dispatchSyscall(syscall)
	code = int(syscall.R0)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolinePollOpen(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	desc := syscallabi.NewValueView(syscall.Ptr1.(*syscallabi.PollDesc))
	code := syscall.OS.(*LinuxOS).PollOpen(fd, desc)
	syscall.R0 = uintptr(code)
	syscall.Complete()
}

//go:norace
func SyscallSysAccept4(s int, rsa *RawSockaddrAny, addrlen *Socklen, flags int) (fd int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysAccept4
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Ptr1 = rsa
	syscall.Ptr2 = addrlen
	syscall.Int3 = uintptr(flags)
	linuxOS.dispatchSyscall(syscall)
	fd = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	syscall.Ptr2 = nil
	return
}

//go:norace
func trampolineSysAccept4(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	rsa := syscallabi.NewValueView(syscall.Ptr1.(*RawSockaddrAny))
	addrlen := syscallabi.NewValueView(syscall.Ptr2.(*Socklen))
	flags := int(syscall.Int3)
	fd, err := syscall.OS.(*LinuxOS).SysAccept4(s, rsa, addrlen, flags)
	syscall.R0 = uintptr(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysBind(s int, addr unsafe.Pointer, addrlen Socklen) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysBind
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Ptr1 = addr
	syscall.Int2 = uintptr(addrlen)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysBind(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	addr := syscall.Ptr1.(unsafe.Pointer)
	addrlen := Socklen(syscall.Int2)
	err := syscall.OS.(*LinuxOS).SysBind(s, addr, addrlen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysChdir(path string) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysChdir
	syscall.OS = linuxOS
	syscall.Ptr0 = path
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr0 = nil
	return
}

//go:norace
func trampolineSysChdir(syscall *syscallabi.Syscall) {
	path := syscall.Ptr0.(string)
	err := syscall.OS.(*LinuxOS).SysChdir(path)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysClose(fd int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysClose
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysClose(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	err := syscall.OS.(*LinuxOS).SysClose(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysConnect(s int, addr unsafe.Pointer, addrlen Socklen) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysConnect
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Ptr1 = addr
	syscall.Int2 = uintptr(addrlen)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysConnect(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	addr := syscall.Ptr1.(unsafe.Pointer)
	addrlen := Socklen(syscall.Int2)
	err := syscall.OS.(*LinuxOS).SysConnect(s, addr, addrlen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFallocate(fd int, mode uint32, off int64, len int64) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFallocate
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(mode)
	syscall.Int2 = uintptr(off)
	syscall.Int3 = uintptr(len)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFallocate(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	mode := uint32(syscall.Int1)
	off := int64(syscall.Int2)
	len := int64(syscall.Int3)
	err := syscall.OS.(*LinuxOS).SysFallocate(fd, mode, off, len)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFcntl(fd int, cmd int, arg int) (val int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFcntl
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(cmd)
	syscall.Int2 = uintptr(arg)
	linuxOS.dispatchSyscall(syscall)
	val = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFcntl(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	cmd := int(syscall.Int1)
	arg := int(syscall.Int2)
	val, err := syscall.OS.(*LinuxOS).SysFcntl(fd, cmd, arg)
	syscall.R0 = uintptr(val)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFdatasync(fd int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFdatasync
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFdatasync(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	err := syscall.OS.(*LinuxOS).SysFdatasync(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFlock(fd int, how int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFlock
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(how)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFlock(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	how := int(syscall.Int1)
	err := syscall.OS.(*LinuxOS).SysFlock(fd, how)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFstat(fd int, stat *Stat_t) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFstat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1 = stat
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysFstat(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	stat := syscallabi.NewValueView(syscall.Ptr1.(*Stat_t))
	err := syscall.OS.(*LinuxOS).SysFstat(fd, stat)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFstatat(dirfd int, path string, stat *Stat_t, flags int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFstatat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(dirfd)
	syscall.Ptr1 = path
	syscall.Ptr2 = stat
	syscall.Int3 = uintptr(flags)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	syscall.Ptr2 = nil
	return
}

//go:norace
func trampolineSysFstatat(syscall *syscallabi.Syscall) {
	dirfd := int(syscall.Int0)
	path := syscall.Ptr1.(string)
	stat := syscallabi.NewValueView(syscall.Ptr2.(*Stat_t))
	flags := int(syscall.Int3)
	err := syscall.OS.(*LinuxOS).SysFstatat(dirfd, path, stat, flags)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFsync(fd int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFsync
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFsync(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	err := syscall.OS.(*LinuxOS).SysFsync(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysFtruncate(fd int, length int64) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFtruncate
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(length)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysFtruncate(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	length := int64(syscall.Int1)
	err := syscall.OS.(*LinuxOS).SysFtruncate(fd, length)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetcwd(buf []byte) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetcwd
	syscall.OS = linuxOS
	syscall.Ptr0, syscall.Int0 = unsafe.SliceData(buf), uintptr(len(buf))
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr0 = nil
	return
}

//go:norace
func trampolineSysGetcwd(syscall *syscallabi.Syscall) {
	buf := syscallabi.NewSliceView(syscall.Ptr0.(*byte), syscall.Int0)
	n, err := syscall.OS.(*LinuxOS).SysGetcwd(buf)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetdents64(fd int, buf []byte) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetdents64
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1, syscall.Int1 = unsafe.SliceData(buf), uintptr(len(buf))
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysGetdents64(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	buf := syscallabi.NewSliceView(syscall.Ptr1.(*byte), syscall.Int1)
	n, err := syscall.OS.(*LinuxOS).SysGetdents64(fd, buf)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetpeername(fd int, rsa *RawSockaddrAny, addrlen *Socklen) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetpeername
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1 = rsa
	syscall.Ptr2 = addrlen
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	syscall.Ptr2 = nil
	return
}

//go:norace
func trampolineSysGetpeername(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	rsa := syscallabi.NewValueView(syscall.Ptr1.(*RawSockaddrAny))
	addrlen := syscallabi.NewValueView(syscall.Ptr2.(*Socklen))
	err := syscall.OS.(*LinuxOS).SysGetpeername(fd, rsa, addrlen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetpid() (pid int) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetpid
	syscall.OS = linuxOS
	linuxOS.dispatchSyscall(syscall)
	pid = int(syscall.R0)
	return
}

//go:norace
func trampolineSysGetpid(syscall *syscallabi.Syscall) {
	pid := syscall.OS.(*LinuxOS).SysGetpid()
	syscall.R0 = uintptr(pid)
	syscall.Complete()
}

//go:norace
func SyscallSysGetrandom(buf []byte, flags int) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetrandom
	syscall.OS = linuxOS
	syscall.Ptr0, syscall.Int0 = unsafe.SliceData(buf), uintptr(len(buf))
	syscall.Int1 = uintptr(flags)
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr0 = nil
	return
}

//go:norace
func trampolineSysGetrandom(syscall *syscallabi.Syscall) {
	buf := syscallabi.NewSliceView(syscall.Ptr0.(*byte), syscall.Int0)
	flags := int(syscall.Int1)
	n, err := syscall.OS.(*LinuxOS).SysGetrandom(buf, flags)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetsockname(fd int, rsa *RawSockaddrAny, addrlen *Socklen) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetsockname
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1 = rsa
	syscall.Ptr2 = addrlen
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	syscall.Ptr2 = nil
	return
}

//go:norace
func trampolineSysGetsockname(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	rsa := syscallabi.NewValueView(syscall.Ptr1.(*RawSockaddrAny))
	addrlen := syscallabi.NewValueView(syscall.Ptr2.(*Socklen))
	err := syscall.OS.(*LinuxOS).SysGetsockname(fd, rsa, addrlen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysGetsockopt(s int, level int, name int, val unsafe.Pointer, vallen *Socklen) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysGetsockopt
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Int1 = uintptr(level)
	syscall.Int2 = uintptr(name)
	syscall.Ptr3 = val
	syscall.Ptr4 = vallen
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr3 = nil
	syscall.Ptr4 = nil
	return
}

//go:norace
func trampolineSysGetsockopt(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	level := int(syscall.Int1)
	name := int(syscall.Int2)
	val := syscall.Ptr3.(unsafe.Pointer)
	vallen := syscallabi.NewValueView(syscall.Ptr4.(*Socklen))
	err := syscall.OS.(*LinuxOS).SysGetsockopt(s, level, name, val, vallen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysListen(s int, n int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysListen
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Int1 = uintptr(n)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysListen(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	n := int(syscall.Int1)
	err := syscall.OS.(*LinuxOS).SysListen(s, n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysLseek(fd int, offset int64, whence int) (off int64, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysLseek
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(offset)
	syscall.Int2 = uintptr(whence)
	linuxOS.dispatchSyscall(syscall)
	off = int64(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysLseek(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	offset := int64(syscall.Int1)
	whence := int(syscall.Int2)
	off, err := syscall.OS.(*LinuxOS).SysLseek(fd, offset, whence)
	syscall.R0 = uintptr(off)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysMadvise(b []byte, advice int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysMadvise
	syscall.OS = linuxOS
	syscall.Ptr0, syscall.Int0 = unsafe.SliceData(b), uintptr(len(b))
	syscall.Int1 = uintptr(advice)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr0 = nil
	return
}

//go:norace
func trampolineSysMadvise(syscall *syscallabi.Syscall) {
	b := syscallabi.NewSliceView(syscall.Ptr0.(*byte), syscall.Int0)
	advice := int(syscall.Int1)
	err := syscall.OS.(*LinuxOS).SysMadvise(b, advice)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysMkdirat(dirfd int, path string, mode uint32) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysMkdirat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(dirfd)
	syscall.Ptr1 = path
	syscall.Int2 = uintptr(mode)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysMkdirat(syscall *syscallabi.Syscall) {
	dirfd := int(syscall.Int0)
	path := syscall.Ptr1.(string)
	mode := uint32(syscall.Int2)
	err := syscall.OS.(*LinuxOS).SysMkdirat(dirfd, path, mode)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysMmap(addr uintptr, length uintptr, prot int, flags int, fd int, offset int64) (xaddr uintptr, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysMmap
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(addr)
	syscall.Int1 = uintptr(length)
	syscall.Int2 = uintptr(prot)
	syscall.Int3 = uintptr(flags)
	syscall.Int4 = uintptr(fd)
	syscall.Int5 = uintptr(offset)
	linuxOS.dispatchSyscall(syscall)
	xaddr = uintptr(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysMmap(syscall *syscallabi.Syscall) {
	addr := uintptr(syscall.Int0)
	length := uintptr(syscall.Int1)
	prot := int(syscall.Int2)
	flags := int(syscall.Int3)
	fd := int(syscall.Int4)
	offset := int64(syscall.Int5)
	xaddr, err := syscall.OS.(*LinuxOS).SysMmap(addr, length, prot, flags, fd, offset)
	syscall.R0 = uintptr(xaddr)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysMunmap(addr uintptr, length uintptr) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysMunmap
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(addr)
	syscall.Int1 = uintptr(length)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysMunmap(syscall *syscallabi.Syscall) {
	addr := uintptr(syscall.Int0)
	length := uintptr(syscall.Int1)
	err := syscall.OS.(*LinuxOS).SysMunmap(addr, length)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysOpenat(dirfd int, path string, flags int, mode uint32) (fd int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysOpenat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(dirfd)
	syscall.Ptr1 = path
	syscall.Int2 = uintptr(flags)
	syscall.Int3 = uintptr(mode)
	linuxOS.dispatchSyscall(syscall)
	fd = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysOpenat(syscall *syscallabi.Syscall) {
	dirfd := int(syscall.Int0)
	path := syscall.Ptr1.(string)
	flags := int(syscall.Int2)
	mode := uint32(syscall.Int3)
	fd, err := syscall.OS.(*LinuxOS).SysOpenat(dirfd, path, flags, mode)
	syscall.R0 = uintptr(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysPread64(fd int, p []byte, offset int64) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysPread64
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1, syscall.Int1 = unsafe.SliceData(p), uintptr(len(p))
	syscall.Int2 = uintptr(offset)
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysPread64(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	p := syscallabi.NewSliceView(syscall.Ptr1.(*byte), syscall.Int1)
	offset := int64(syscall.Int2)
	n, err := syscall.OS.(*LinuxOS).SysPread64(fd, p, offset)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysPwrite64(fd int, p []byte, offset int64) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysPwrite64
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1, syscall.Int1 = unsafe.SliceData(p), uintptr(len(p))
	syscall.Int2 = uintptr(offset)
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysPwrite64(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	p := syscallabi.NewSliceView(syscall.Ptr1.(*byte), syscall.Int1)
	offset := int64(syscall.Int2)
	n, err := syscall.OS.(*LinuxOS).SysPwrite64(fd, p, offset)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysRead(fd int, p []byte) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysRead
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1, syscall.Int1 = unsafe.SliceData(p), uintptr(len(p))
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysRead(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	p := syscallabi.NewSliceView(syscall.Ptr1.(*byte), syscall.Int1)
	n, err := syscall.OS.(*LinuxOS).SysRead(fd, p)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysRenameat(olddirfd int, oldpath string, newdirfd int, newpath string) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysRenameat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(olddirfd)
	syscall.Ptr1 = oldpath
	syscall.Int2 = uintptr(newdirfd)
	syscall.Ptr3 = newpath
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	syscall.Ptr3 = nil
	return
}

//go:norace
func trampolineSysRenameat(syscall *syscallabi.Syscall) {
	olddirfd := int(syscall.Int0)
	oldpath := syscall.Ptr1.(string)
	newdirfd := int(syscall.Int2)
	newpath := syscall.Ptr3.(string)
	err := syscall.OS.(*LinuxOS).SysRenameat(olddirfd, oldpath, newdirfd, newpath)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysSetsockopt(s int, level int, name int, val unsafe.Pointer, vallen uintptr) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysSetsockopt
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(s)
	syscall.Int1 = uintptr(level)
	syscall.Int2 = uintptr(name)
	syscall.Ptr3 = val
	syscall.Int4 = uintptr(vallen)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr3 = nil
	return
}

//go:norace
func trampolineSysSetsockopt(syscall *syscallabi.Syscall) {
	s := int(syscall.Int0)
	level := int(syscall.Int1)
	name := int(syscall.Int2)
	val := syscall.Ptr3.(unsafe.Pointer)
	vallen := uintptr(syscall.Int4)
	err := syscall.OS.(*LinuxOS).SysSetsockopt(s, level, name, val, vallen)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysSocket(domain int, typ int, proto int) (fd int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysSocket
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(domain)
	syscall.Int1 = uintptr(typ)
	syscall.Int2 = uintptr(proto)
	linuxOS.dispatchSyscall(syscall)
	fd = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	return
}

//go:norace
func trampolineSysSocket(syscall *syscallabi.Syscall) {
	domain := int(syscall.Int0)
	typ := int(syscall.Int1)
	proto := int(syscall.Int2)
	fd, err := syscall.OS.(*LinuxOS).SysSocket(domain, typ, proto)
	syscall.R0 = uintptr(fd)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysUname(buf *Utsname) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysUname
	syscall.OS = linuxOS
	syscall.Ptr0 = buf
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr0 = nil
	return
}

//go:norace
func trampolineSysUname(syscall *syscallabi.Syscall) {
	buf := syscallabi.NewValueView(syscall.Ptr0.(*Utsname))
	err := syscall.OS.(*LinuxOS).SysUname(buf)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysUnlinkat(dirfd int, path string, flags int) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysUnlinkat
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(dirfd)
	syscall.Ptr1 = path
	syscall.Int2 = uintptr(flags)
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysUnlinkat(syscall *syscallabi.Syscall) {
	dirfd := int(syscall.Int0)
	path := syscall.Ptr1.(string)
	flags := int(syscall.Int2)
	err := syscall.OS.(*LinuxOS).SysUnlinkat(dirfd, path, flags)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

//go:norace
func SyscallSysWrite(fd int, p []byte) (n int, err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysWrite
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Ptr1, syscall.Int1 = unsafe.SliceData(p), uintptr(len(p))
	linuxOS.dispatchSyscall(syscall)
	n = int(syscall.R0)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr1 = nil
	return
}

//go:norace
func trampolineSysWrite(syscall *syscallabi.Syscall) {
	fd := int(syscall.Int0)
	p := syscallabi.NewSliceView(syscall.Ptr1.(*byte), syscall.Int1)
	n, err := syscall.OS.(*LinuxOS).SysWrite(fd, p)
	syscall.R0 = uintptr(n)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}

func IsHandledSyscall(trap uintptr) bool {
	switch trap {
	case unix.SYS_ACCEPT4:
		return true
	case unix.SYS_BIND:
		return true
	case unix.SYS_CHDIR:
		return true
	case unix.SYS_CLOSE:
		return true
	case unix.SYS_CONNECT:
		return true
	case unix.SYS_FALLOCATE:
		return true
	case unix.SYS_FCNTL:
		return true
	case unix.SYS_FDATASYNC:
		return true
	case unix.SYS_FLOCK:
		return true
	case unix.SYS_FSTAT:
		return true
	case unix.SYS_FSTATAT:
		return true
	case unix.SYS_FSYNC:
		return true
	case unix.SYS_FTRUNCATE:
		return true
	case unix.SYS_GETCWD:
		return true
	case unix.SYS_GETDENTS64:
		return true
	case unix.SYS_GETPEERNAME:
		return true
	case unix.SYS_GETPID:
		return true
	case unix.SYS_GETRANDOM:
		return true
	case unix.SYS_GETSOCKNAME:
		return true
	case unix.SYS_GETSOCKOPT:
		return true
	case unix.SYS_LISTEN:
		return true
	case unix.SYS_LSEEK:
		return true
	case unix.SYS_MADVISE:
		return true
	case unix.SYS_MKDIRAT:
		return true
	case unix.SYS_MMAP:
		return true
	case unix.SYS_MUNMAP:
		return true
	case unix.SYS_OPENAT:
		return true
	case unix.SYS_PREAD64:
		return true
	case unix.SYS_PWRITE64:
		return true
	case unix.SYS_READ:
		return true
	case unix.SYS_RENAMEAT:
		return true
	case unix.SYS_SETSOCKOPT:
		return true
	case unix.SYS_SOCKET:
		return true
	case unix.SYS_UNAME:
		return true
	case unix.SYS_UNLINKAT:
		return true
	case unix.SYS_WRITE:
		return true
	default:
		return false
	}
}
