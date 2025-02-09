//go:build linux

package simulation

import (
	"syscall"

	"github.com/kmrgirish/gosim/internal/simulation/syscallabi"
)

// manually implemented system calls...

//

type Flock_t = syscall.Flock_t

// This ought to call the Fcntl wrapped generated by gensyscall,
// but the generated Fcntl wrapper does not take a pointer, and passing a
// uintptr would not be safe.
// This wrapper works, but it does not show in the handled syscalls switch.
//
//go:norace
func SyscallSysFcntlFlock(fd uintptr, cmd int, lk *Flock_t) (err error) {
	syscall := syscallabi.GetGoroutineLocalSyscall()
	syscall.Trampoline = trampolineSysFcntlFlock
	syscall.OS = linuxOS
	syscall.Int0 = uintptr(fd)
	syscall.Int1 = uintptr(cmd)
	syscall.Ptr2 = lk
	linuxOS.dispatchSyscall(syscall)
	err = syscallabi.ErrnoErr(syscall.Errno)
	syscall.Ptr2 = nil
	return
}

//go:norace
func trampolineSysFcntlFlock(syscall *syscallabi.Syscall) {
	fd := uintptr(syscall.Int0)
	cmd := int(syscall.Int1)
	lk := syscallabi.NewValueView(syscall.Ptr2.(*Flock_t))
	err := syscall.OS.(*LinuxOS).SysFcntlFlock(fd, cmd, lk, syscall)
	syscall.Errno = syscallabi.ErrErrno(err)
	syscall.Complete()
}
