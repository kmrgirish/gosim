//go:build linux

package go123

import "syscall"

func InternalSyscallUnix_fcntl(fd int32, cmd int32, args int32) (int32, int32) {
	return 0, int32(syscall.ENOSYS)
}

func InternalSyscallUnix_GetRandom(p []byte, flags uintptr) (n int, err error) {
	return GolangOrgXSysUnix_Getrandom(p, int(flags))
}
