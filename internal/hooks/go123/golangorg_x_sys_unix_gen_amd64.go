//go:build linux

// Code generated by gensyscall. DO NOT EDIT.
package go123

import (
	"unsafe"

	"github.com/kmrgirish/gosim/internal/simulation"
)

// prevent unused imports
var _ unsafe.Pointer

func GolangOrgXSysUnix_accept4(s int, rsa *simulation.RawSockaddrAny, addrlen *simulation.Socklen, flags int) (fd int, err error) {
	return simulation.SyscallSysAccept4(s, rsa, addrlen, flags)
}

func GolangOrgXSysUnix_bind(s int, addr unsafe.Pointer, addrlen simulation.Socklen) (err error) {
	return simulation.SyscallSysBind(s, addr, addrlen)
}

func GolangOrgXSysUnix_Chdir(path string) (err error) {
	return simulation.SyscallSysChdir(path)
}

func GolangOrgXSysUnix_Close(fd int) (err error) {
	return simulation.SyscallSysClose(fd)
}

func GolangOrgXSysUnix_connect(s int, addr unsafe.Pointer, addrlen simulation.Socklen) (err error) {
	return simulation.SyscallSysConnect(s, addr, addrlen)
}

func GolangOrgXSysUnix_Fallocate(fd int, mode uint32, off int64, len int64) (err error) {
	return simulation.SyscallSysFallocate(fd, mode, off, len)
}

func GolangOrgXSysUnix_Fdatasync(fd int) (err error) {
	return simulation.SyscallSysFdatasync(fd)
}

func GolangOrgXSysUnix_Flock(fd int, how int) (err error) {
	return simulation.SyscallSysFlock(fd, how)
}

func GolangOrgXSysUnix_Fstat(fd int, stat *simulation.Stat_t) (err error) {
	return simulation.SyscallSysFstat(fd, stat)
}

func GolangOrgXSysUnix_Fstatat(dirfd int, path string, stat *simulation.Stat_t, flags int) (err error) {
	return simulation.SyscallSysNewfstatat(dirfd, path, stat, flags)
}

func GolangOrgXSysUnix_Fsync(fd int) (err error) {
	return simulation.SyscallSysFsync(fd)
}

func GolangOrgXSysUnix_Ftruncate(fd int, length int64) (err error) {
	return simulation.SyscallSysFtruncate(fd, length)
}

func GolangOrgXSysUnix_Getcwd(buf []byte) (n int, err error) {
	return simulation.SyscallSysGetcwd(buf)
}

func GolangOrgXSysUnix_Getdents(fd int, buf []byte) (n int, err error) {
	return simulation.SyscallSysGetdents64(fd, buf)
}

func GolangOrgXSysUnix_getpeername(fd int, rsa *simulation.RawSockaddrAny, addrlen *simulation.Socklen) (err error) {
	return simulation.SyscallSysGetpeername(fd, rsa, addrlen)
}

func GolangOrgXSysUnix_Getpid() (pid int) {
	return simulation.SyscallSysGetpid()
}

func GolangOrgXSysUnix_Getrandom(buf []byte, flags int) (n int, err error) {
	return simulation.SyscallSysGetrandom(buf, flags)
}

func GolangOrgXSysUnix_getsockname(fd int, rsa *simulation.RawSockaddrAny, addrlen *simulation.Socklen) (err error) {
	return simulation.SyscallSysGetsockname(fd, rsa, addrlen)
}

func GolangOrgXSysUnix_getsockopt(s int, level int, name int, val unsafe.Pointer, vallen *simulation.Socklen) (err error) {
	return simulation.SyscallSysGetsockopt(s, level, name, val, vallen)
}

func GolangOrgXSysUnix_Listen(s int, n int) (err error) {
	return simulation.SyscallSysListen(s, n)
}

func GolangOrgXSysUnix_Madvise(b []byte, advice int) (err error) {
	return simulation.SyscallSysMadvise(b, advice)
}

func GolangOrgXSysUnix_Mkdirat(dirfd int, path string, mode uint32) (err error) {
	return simulation.SyscallSysMkdirat(dirfd, path, mode)
}

func GolangOrgXSysUnix_mmap(addr uintptr, length uintptr, prot int, flags int, fd int, offset int64) (xaddr uintptr, err error) {
	return simulation.SyscallSysMmap(addr, length, prot, flags, fd, offset)
}

func GolangOrgXSysUnix_munmap(addr uintptr, length uintptr) (err error) {
	return simulation.SyscallSysMunmap(addr, length)
}

func GolangOrgXSysUnix_openat(dirfd int, path string, flags int, mode uint32) (fd int, err error) {
	return simulation.SyscallSysOpenat(dirfd, path, flags, mode)
}

func GolangOrgXSysUnix_pread(fd int, p []byte, offset int64) (n int, err error) {
	return simulation.SyscallSysPread64(fd, p, offset)
}

func GolangOrgXSysUnix_pwrite(fd int, p []byte, offset int64) (n int, err error) {
	return simulation.SyscallSysPwrite64(fd, p, offset)
}

func GolangOrgXSysUnix_read(fd int, p []byte) (n int, err error) {
	return simulation.SyscallSysRead(fd, p)
}

func GolangOrgXSysUnix_Renameat(olddirfd int, oldpath string, newdirfd int, newpath string) (err error) {
	return simulation.SyscallSysRenameat(olddirfd, oldpath, newdirfd, newpath)
}

func GolangOrgXSysUnix_Seek(fd int, offset int64, whence int) (off int64, err error) {
	return simulation.SyscallSysLseek(fd, offset, whence)
}

func GolangOrgXSysUnix_setsockopt(s int, level int, name int, val unsafe.Pointer, vallen uintptr) (err error) {
	return simulation.SyscallSysSetsockopt(s, level, name, val, vallen)
}

func GolangOrgXSysUnix_socket(domain int, typ int, proto int) (fd int, err error) {
	return simulation.SyscallSysSocket(domain, typ, proto)
}

func GolangOrgXSysUnix_Uname(buf *simulation.Utsname) (err error) {
	return simulation.SyscallSysUname(buf)
}

func GolangOrgXSysUnix_Unlinkat(dirfd int, path string, flags int) (err error) {
	return simulation.SyscallSysUnlinkat(dirfd, path, flags)
}

func GolangOrgXSysUnix_write(fd int, p []byte) (n int, err error) {
	return simulation.SyscallSysWrite(fd, p)
}
