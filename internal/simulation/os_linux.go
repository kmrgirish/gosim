//go:build linux

package simulation

import (
	"encoding/binary"
	"errors"
	"log"
	"log/slog"
	"math/rand"
	"net/netip"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/jellevandenhooff/gosim/gosimruntime"
	"github.com/jellevandenhooff/gosim/internal/simulation/fs"
	"github.com/jellevandenhooff/gosim/internal/simulation/network"
	"github.com/jellevandenhooff/gosim/internal/simulation/syscallabi"
)

// LinuxOS implements gosim's versions of Linux system calls.
type LinuxOS struct {
	dispatcher syscallabi.Dispatcher

	simulation *Simulation

	machine *Machine

	mu       sync.Mutex
	shutdown bool

	files  map[int]interface{}
	nextFd int
}

func NewLinuxOS(simulation *Simulation, machine *Machine, dispatcher syscallabi.Dispatcher) *LinuxOS {
	return &LinuxOS{
		dispatcher: dispatcher,

		simulation: simulation,

		machine: machine,

		files:  make(map[int]interface{}),
		nextFd: 5,
	}
}

func (l *LinuxOS) doShutdown() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.shutdown = true

	// nil out pointers to catch any use after shutdown
	l.machine = nil
	l.files = nil
}

// XXX: var causes init issues????
const logEnabled = false

func logf(fmt string, args ...any) {
	if logEnabled {
		log.Printf(fmt, args...)
	}
}

// used to be...
// //go:nocheckptr
// //go:uintptrescapes
func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno) {
	if IsHandledSyscall(trap) {
		// complain loudly if syscall that we support arrives
		// here instead of through our renames
		panic("syscall should have been rewritten somewhere")
	}

	logf("unknown=%d %d %d %d %d %d %d", trap, a1, a2, a3, a4, a5, a6)

	return 0, 0, syscall.ENOSYS
}

func (l *LinuxOS) allocFd() int {
	fd := l.nextFd
	l.nextFd++
	return fd
}

func (l *LinuxOS) PollOpen(fd int, desc syscallabi.ValueView[syscallabi.PollDesc]) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		// XXX: what do we return here? shouldn't matter?
		return 0
	}

	f := l.files[fd]
	if socket, ok := f.(*Socket); ok {
		if socket.Stream != nil {
			l.machine.netstack.RegisterStreamPoller((*syscallabi.PollDesc)(desc.UnsafePointer()), socket.Stream)
		}
		if socket.Listener != nil {
			l.machine.netstack.RegisterListenerPoller((*syscallabi.PollDesc)(desc.UnsafePointer()), socket.Listener)
		}
	}
	return 0
}

func (l *LinuxOS) PollClose(fd int, desc syscallabi.ValueView[syscallabi.PollDesc]) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		// XXX: what do we return here? shouldn't matter?
		return 0
	}

	f := l.files[fd]
	if socket, ok := f.(*Socket); ok {
		if socket.Stream != nil {
			l.machine.netstack.DeregisterStreamPoller((*syscallabi.PollDesc)(desc.UnsafePointer()), socket.Stream)
		}
		if socket.Listener != nil {
			l.machine.netstack.DeregisterListenerPoller((*syscallabi.PollDesc)(desc.UnsafePointer()), socket.Listener)
		}
	}
	return 0
}

type (
	RawSockaddrAny = syscall.RawSockaddrAny
	Utsname        = syscall.Utsname
	Stat_t         = syscall.Stat_t
)

type Socklen uint32

const (
	_AT_FDCWD     = -0x64
	_AT_REMOVEDIR = 0x200
)

// TODO: make functions instead return Errno again?

// for disk:
// to sim
// - random op errors
// - latency
// - writes get split in parts

// behavior tests
// - assert that behavior shows up in (some) runs
// - assert that behavior never shows up
// - assert that we have all possible behavior (exact)

// brokenness that might happen:
// - file entries are not are persisted until fsync on the directory [done]
// - writes to a file might be reordered until fsync [done]
// - writes to different files might be reordered [done]
// - an appended-to file might have zeroes [done]
// - after truncate a file might still have old data afterwards (zeroes not guaranteed)

// XXX:
// - (later:) alert on cross-machine interactions (eg. chan, memory, ...?)

type OsFile struct {
	inode               int
	name                string
	pos                 int64
	flagRead, flagWrite bool

	didReaddir bool // XXX JANK
}

type Socket struct {
	IsBound   bool
	BoundAddr netip.AddrPort

	Listener *network.Listener
	Stream   *network.Stream

	RecvBuffer []byte

	// pending socket
	// tcp listener
	// file
	// tcp conn
}

// persist dir:
// all links into this dir we persist

func (l *LinuxOS) SysOpenat(dirfd int, path string, flags int, mode uint32) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	if dirfd != _AT_FDCWD {
		return 0, syscall.EINVAL // XXX?
	}

	// just get rid of this
	flags &= ^syscall.O_CLOEXEC

	logf("openat %d %s %d %d", dirfd, path, flags, mode)

	// TODO: some rules about paths; component length; total length; allow characters?
	// TODO: check mode

	const flagSupported = os.O_RDONLY | os.O_WRONLY | os.O_RDWR |
		os.O_CREATE | os.O_EXCL | os.O_TRUNC

	if flags&(^flagSupported) != 0 {
		return 0, syscall.EINVAL // XXX?
	}

	var flagRead, flagWrite bool
	if flags&os.O_WRONLY != 0 {
		if flags&(os.O_RDWR) != 0 {
			return 0, syscall.EINVAL // XXX?
		}
		flagWrite = true
	} else if flags&os.O_RDWR != 0 {
		flagRead = true
		flagWrite = true
	} else {
		flagRead = true
	}

	inode, err := l.machine.filesystem.OpenFile(path, flags)
	if err != nil {
		if err == syscall.ENOENT {
			return 0, syscall.ENOENT
		}
		return 0, syscall.EINVAL // XXX
	}

	// TODO: add reference here? or in OpenFile? help this is strange.

	// TODO: this pos is wrong for writing maybe? what should happen when you
	// open a file that already exists and start writing to it?
	f := &OsFile{
		name:      path,
		inode:     inode,
		pos:       0,
		flagRead:  flagRead,
		flagWrite: flagWrite,
	}

	fd := l.allocFd()
	l.files[fd] = f

	return fd, nil
}

func (l *LinuxOS) SysRenameat(olddirfd int, oldpath string, newdirfd int, newpath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("renameat %d %s %d %s", olddirfd, oldpath, newdirfd, newpath)

	if olddirfd != _AT_FDCWD {
		return syscall.EINVAL // XXX?
	}

	if newdirfd != _AT_FDCWD {
		return syscall.EINVAL // XXX?
	}

	if err := l.machine.filesystem.Rename(oldpath, newpath); err != nil {
		if err == syscall.ENOENT {
			return syscall.ENOENT
		}
		return syscall.EINVAL // XXX?
	}

	return nil
}

func (l *LinuxOS) SysGetdents64(fd int, data syscallabi.ByteSliceView) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	logf("getdents64 %d %d", fd, data.Len())

	fdInternal, ok := l.files[fd]
	if !ok {
		return 0, syscall.EBADFD
	}

	f, ok := fdInternal.(*OsFile)
	if !ok {
		return 0, syscall.ENOTDIR
	}

	if f.didReaddir {
		return 0, nil
	}

	entries, err := l.machine.filesystem.ReadDir(f.name)
	if err != nil {
		return 0, syscall.EINVAL // XXX?
	}

	var localbuffer [4096]byte

	retn := 0
	for _, entry := range entries {
		name := entry.Name

		inode := int64(1)
		offset := int64(1)
		reclen := int(19) + len(name) + 1

		if reclen > len(localbuffer) {
			panic("help")
		}
		if retn+reclen > data.Len() {
			break
		}

		buffer := localbuffer[:reclen]
		typ := syscall.DT_REG
		if entry.IsDir {
			typ = syscall.DT_DIR
		}

		binary.LittleEndian.PutUint64(buffer[0:8], uint64(inode))
		binary.LittleEndian.PutUint64(buffer[8:16], uint64(offset))
		binary.LittleEndian.PutUint16(buffer[16:18], uint16(reclen))
		buffer[18] = byte(typ)
		copy(buffer[19:], name)
		buffer[19+len(name)] = 0

		data.Slice(retn, retn+reclen).Write(buffer)
		retn += reclen
	}

	logf("getdents64 %d %d", fd, data.Len() /*, data*/)

	f.didReaddir = true

	return retn, nil
}

func (l *LinuxOS) SysWrite(fd int, data syscallabi.ByteSliceView) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	if fd == syscall.Stdout || fd == syscall.Stderr {
		buf := make([]byte, data.Len())
		data.Read(buf)
		// TODO: use a custom machine here?

		var source string
		if fd == syscall.Stdout {
			source = "stdout"
		} else {
			source = "stderr"
		}

		slog.Info(string(buf), "method", source, "from", l.machine.label)
		return data.Len(), nil
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		logf("write %d badfd", fd)
		return 0, syscall.EBADFD
	}

	switch f := fdInternal.(type) {
	case *OsFile:
		if !f.flagWrite {
			return 0, syscall.EBADFD
		}

		retn := l.machine.filesystem.Write(f.inode, f.pos, data)
		f.pos += int64(retn)

		logf("write %d %d %q", fd, data.Len(), data)
		return retn, nil

	case *Socket:
		if f.Stream == nil {
			return 0, syscall.EBADFD
		}

		retn, err := l.machine.netstack.StreamSend(f.Stream, data)
		if err != nil {
			if err == syscall.EWOULDBLOCK {
				return retn, syscall.EWOULDBLOCK
			}
			if err == network.ErrStreamClosed {
				return retn, syscall.EPIPE
			}
			return retn, syscall.EBADFD
		}

		logf("write %d %d %q", fd, data.Len(), data)
		return retn, nil

	default:
		return 0, syscall.EBADFD
	}
}

func (l *LinuxOS) SysRead(fd int, data syscallabi.ByteSliceView) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return 0, syscall.EBADFD
	}

	switch f := fdInternal.(type) {
	case *OsFile:
		if !f.flagRead {
			return 0, syscall.EBADFD // XXX??
		}

		retn := l.machine.filesystem.Read(f.inode, f.pos, data)
		f.pos += int64(retn)

		logf("read %d %d", fd, data.Len() /*, data[:retn]*/)

		return retn, nil

	case *Socket:
		if f.Stream == nil {
			return 0, syscall.EBADFD
		}

		retn, err := l.machine.netstack.StreamRecv(f.Stream, data)
		if err != nil {
			if err == syscall.EWOULDBLOCK {
				return 0, syscall.EWOULDBLOCK
			}
			if err == network.ErrStreamClosed {
				return 0, syscall.EPIPE
			}
			return 0, syscall.EBADFD
		}

		logf("read %d %d", fd, data.Len() /*, data[:retn]*/)

		return retn, nil

	default:
		return 0, syscall.EBADFD
	}
}

func (l *LinuxOS) SysPwrite64(fd int, data syscallabi.ByteSliceView, offset int64) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return 0, syscall.EBADFD
	}

	f, ok := fdInternal.(*OsFile)
	if !ok {
		return 0, syscall.EBADFD
	}

	if !f.flagWrite {
		return 0, syscall.EBADFD
	}

	retn := l.machine.filesystem.Write(f.inode, offset, data)

	logf("writeat %d %d %q %d", fd, retn, data, offset)

	return retn, nil
}

func (l *LinuxOS) SysPread64(fd int, data syscallabi.ByteSliceView, offset int64) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return 0, syscall.EBADFD
	}

	f, ok := fdInternal.(*OsFile)
	if !ok {
		return 0, syscall.EBADFD
	}

	if !f.flagRead {
		return 0, syscall.EBADFD // XXX??
	}

	retn := l.machine.filesystem.Read(f.inode, offset, data)

	logf("readat %d %d %d", fd, data.Len() /*, data[:retn]*/, offset)

	return retn, nil
}

func (l *LinuxOS) SysFsync(fd int) error {
	// XXX: janky way to crash on sync
	l.machine.sometimesCrashOnSyncMu.Lock()
	maybeCrash := l.machine.sometimesCrashOnSync
	l.machine.sometimesCrashOnSyncMu.Unlock()
	if maybeCrash && rand.Intn(100) > 90 {
		if rand.Int()&1 == 0 {
			// XXX: include fd's file somehow???
			slog.Info("crashing machine before sync", "machine", l.machine.label)
			l.simulation.mu.Lock()
			defer l.simulation.mu.Unlock()
			l.simulation.stopMachine(l.machine, false)
			return syscall.EINTR // XXX whatever this should never be seen
		} else {
			defer func() {
				slog.Info("crashing machine after sync", "machine", l.machine.label)
				l.simulation.mu.Lock()
				defer l.simulation.mu.Unlock()
				l.simulation.stopMachine(l.machine, false)
			}()
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	f, ok := fdInternal.(*OsFile)
	if !ok {
		return syscall.EBADFD
	}

	l.machine.filesystem.Sync(f.inode)

	logf("fsync %d", fd)

	return nil
}

func fillStat(view syscallabi.ValueView[syscall.Stat_t], in fs.StatResp) {
	var out syscall.Stat_t
	out.Ino = uint64(in.Inode)
	if in.IsDir {
		out.Mode = syscall.S_IFDIR
	} else {
		out.Mode = syscall.S_IFREG
	}
	view.Set(out)
}

func (l *LinuxOS) SysFstat(fd int, statBuf syscallabi.ValueView[syscall.Stat_t]) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("fstat %d", fd)

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	// XXX: handle dir
	f, ok := fdInternal.(*OsFile)
	if !ok {
		return syscall.EBADFD
	}

	fsStat, err := l.machine.filesystem.Statfd(f.inode)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return syscall.ENOENT
		}
		return syscall.EINVAL // XXX?
	}

	fillStat(statBuf, fsStat)

	return nil
}

// arm64
func (l *LinuxOS) SysFstatat(dirfd int, path string, statBuf syscallabi.ValueView[syscall.Stat_t], flags int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	if dirfd != _AT_FDCWD {
		return syscall.EINVAL // XXX?
	}

	if flags != 0 {
		return syscall.EINVAL // XXX?
	}

	logf("fstatat %d %s %d", dirfd, path, flags)

	fsStat, err := l.machine.filesystem.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return syscall.ENOENT
		}
		return syscall.EINVAL // XXX?
	}

	fillStat(statBuf, fsStat)

	return nil
}

// amd64
func (l *LinuxOS) SysNewfstatat(dirfd int, path string, statBuf syscallabi.ValueView[syscall.Stat_t], flags int) error {
	return l.SysFstatat(dirfd, path, statBuf, flags)
}

func (l *LinuxOS) SysUnlinkat(dirfd int, path string, flags int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	if dirfd != _AT_FDCWD {
		return syscall.EINVAL // XXX?
	}

	logf("unlinkat %d %s %d", dirfd, path, flags)

	switch flags {
	case 0:
		if err := l.machine.filesystem.Remove(path); err != nil {
			if err == syscall.ENOENT {
				return syscall.ENOENT
			}
			return syscall.EINVAL
		}
		return nil
	case _AT_REMOVEDIR:
		// XXX: should special case dir vs file and also ENOTDIR
		if err := l.machine.filesystem.Remove(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return syscall.ENOENT
			}
			return syscall.EINVAL
		}

		return nil

	default:
		return syscall.EINVAL
	}
}

func (l *LinuxOS) SysFtruncate(fd int, n int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	f, ok := fdInternal.(*OsFile)
	if !ok {
		return syscall.EBADFD
	}

	if !f.flagWrite {
		return syscall.EBADFD // XXX?
	}

	l.machine.filesystem.Truncate(f.inode, int(n))

	logf("truncate %d %d", fd, n)

	return nil
}

func (l *LinuxOS) SysClose(fd int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("close %d", fd)

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	switch fdInternal := fdInternal.(type) {
	case *OsFile:
		if err := l.machine.filesystem.CloseFile(fdInternal.inode); err != nil {
			// XXX?
			return syscall.EBADFD
		}

	case *Socket:
		// TBD
		// XXX: wowwwwwww we should get this one. better send a close!

	default:
		return syscall.EBADFD
	}

	delete(l.files, fd)

	return nil
}

func (l *LinuxOS) SysSocket(net, flags, proto int) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	// handle sock_posix func (p *ipStackCapabilities) probe()

	logf("socket %d %d %d", net, flags, proto)

	// ipv4 close on exec non blocking tcp streams ONLY
	if net != syscall.AF_INET {
		return 0, syscall.EINVAL // XXX?
	}
	if flags != syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK {
		return 0, syscall.EINVAL // XXX?
	}
	if proto != 0 { // XXX: no ipv6
		return 0, syscall.EINVAL // XXX?
	}

	fd := l.allocFd()
	l.files[fd] = &Socket{}

	return fd, nil
}

func (l *LinuxOS) SysBind(fd int, addrPtr unsafe.Pointer, addrlen Socklen) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	if sock.IsBound {
		return syscall.EINVAL
	}

	addr, err := readAddr(addrPtr, addrlen)
	if err != 0 {
		return err
	}

	logf("bind %d %s", fd, addr)

	switch {
	case addr.Addr().Is4():
	default:
		return syscall.EINVAL
	}

	sock.IsBound = true
	sock.BoundAddr = addr
	return nil
}

func (l *LinuxOS) SysListen(fd, backlog int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	if !sock.IsBound {
		return syscall.EBADFD
	}

	if sock.Listener != nil {
		return syscall.EBADFD
	}

	// XXX
	listener, err := l.machine.netstack.OpenListener(sock.BoundAddr.Port())
	if err != nil {
		return syscall.EINVAL
	}
	sock.Listener = listener

	logf("listen %d %d", fd, backlog)

	return nil
}

func readAddr(addr unsafe.Pointer, addrlen Socklen) (netip.AddrPort, syscall.Errno) {
	rsa := syscallabi.NewValueView((*syscall.RawSockaddr)(addr))

	// XXX: check addrlen here?
	switch rsa.Get().Family {
	case syscall.AF_INET:
		// AF_INET
		if addrlen != syscall.SizeofSockaddrInet4 {
			return netip.AddrPort{}, syscall.EINVAL
		}
		rsa4 := syscallabi.NewValueView((*syscall.RawSockaddrInet4)(addr)).Get()
		ip := netip.AddrFrom4(rsa4.Addr)
		// XXX: HELP
		portBytes := (*[2]byte)(unsafe.Pointer(&rsa4.Port))
		port := uint16(int(portBytes[0])<<8 + int(portBytes[1]))
		for i := 0; i < 8; i++ {
			if rsa4.Zero[i] != 0 {
				return netip.AddrPort{}, syscall.EINVAL
			}
		}
		return netip.AddrPortFrom(ip, port), 0

	default:
		return netip.AddrPort{}, syscall.EINVAL
	}
}

func writeAddr(outbuf syscallabi.ValueView[RawSockaddrAny], outLen syscallabi.ValueView[Socklen], addr netip.AddrPort) syscall.Errno {
	switch {
	case addr.Addr().Is4():
		outbuf := syscallabi.NewValueView((*syscall.RawSockaddrInet4)(outbuf.UnsafePointer()))
		addrPort := addr.Port()
		portBytes := (*[2]byte)(unsafe.Pointer(&addrPort))
		port := uint16(int(portBytes[0])<<8 + int(portBytes[1]))
		outbuf.Set(syscall.RawSockaddrInet4{
			Family: syscall.AF_INET,
			Port:   port,
			Addr:   addr.Addr().As4(),
		})
		outLen.Set(syscall.SizeofSockaddrInet4)
		return 0

	default:
		return syscall.EADDRNOTAVAIL // XXX
	}
}

func (l *LinuxOS) SysAccept4(fd int, rsa syscallabi.ValueView[RawSockaddrAny], len syscallabi.ValueView[Socklen], flags int) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	logf("accept %d %d %d", fd, len.Get(), flags)

	if flags != syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK {
		// XXX: check that flags are CLOEXEC and NONBLOCK (just like sys_socket)
		return 0, syscall.EINVAL // XXX?
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return 0, syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return 0, syscall.ENOTSOCK
	}

	if sock.Listener == nil {
		return 0, syscall.EBADFD
	}

	stream, err := l.machine.netstack.Accept(sock.Listener)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return 0, syscall.EWOULDBLOCK
		}
		return 0, syscall.EBADFD
	}

	newFd := l.allocFd()
	l.files[newFd] = &Socket{
		Stream: stream,
	}

	if err := writeAddr(rsa, len, stream.ID().Dest()); err != 0 {
		return 0, err
	}

	return newFd, nil
}

func (l *LinuxOS) SysGetsockopt(fd int, level int, name int, ptr unsafe.Pointer, outlen syscallabi.ValueView[Socklen]) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("getsockopt %d %d %d %d", fd, level, name, outlen.Get())

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	switch {
	case sock.Stream != nil:
		if level != syscall.SOL_SOCKET || name != syscall.SO_ERROR {
			return syscall.ENOSYS
		}

		if outlen.Get() != 4 {
			return syscall.EINVAL
		}

		// this is used to determine connection status

		status := l.machine.netstack.StreamStatus(sock.Stream)

		var v uint32
		if !status.Open && !status.Closed {
			v = uint32(syscall.EINPROGRESS)
		} else if !status.Open && status.Closed {
			// XXX: disambiguate later
			v = uint32(syscall.ETIMEDOUT)
		} else {
			// XXX: this is where could also say ECONNREFUSED etc?
			// XXX: supposed to clear error afterwards
			v = 0
		}
		syscallabi.NewValueView[uint32]((*uint32)(ptr)).Set(v)
		return nil

	case sock.Listener != nil:
		// XXX: seems bad response but
		return syscall.ENOSYS

	default:
		return syscall.EBADFD
	}
}

func (l *LinuxOS) SysSetsockopt(s int, level int, name int, val unsafe.Pointer, vallen uintptr) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	// XXX: yep
	return nil
}

func (l *LinuxOS) SysGetsockname(fd int, rsa syscallabi.ValueView[RawSockaddrAny], len syscallabi.ValueView[Socklen]) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("getsockname %d %d", fd, len.Get())

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	switch {
	case sock.Stream != nil:
		if err := writeAddr(rsa, len, sock.Stream.ID().Source()); err != 0 {
			return err
		}
		return nil

	case sock.Listener != nil:
		if err := writeAddr(rsa, len, netip.AddrPortFrom(netip.IPv4Unspecified(), 1)); err != 0 { // XXX TBD
			return err
		}
		return nil

	default:
		return syscall.EBADFD
	}
}

func (l *LinuxOS) SysGetpeername(fd int, rsa syscallabi.ValueView[RawSockaddrAny], len syscallabi.ValueView[Socklen]) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	logf("getpeername %d %d", fd, len.Get())

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	if sock.Stream == nil {
		return syscall.EBADFD
	}

	if err := writeAddr(rsa, len, sock.Stream.ID().Dest()); err != 0 {
		return err
	}

	return nil
}

func (l *LinuxOS) SysFcntl(fd int, cmd int, arg int) (val int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	return 0, syscall.ENOSYS
}

func (l *LinuxOS) SysConnect(fd int, addrPtr unsafe.Pointer, addrLen Socklen) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return syscall.EINVAL
	}

	fdInternal, ok := l.files[fd]
	if !ok {
		return syscall.EBADFD
	}

	sock, ok := fdInternal.(*Socket)
	if !ok {
		return syscall.ENOTSOCK
	}

	addr, errno := readAddr(addrPtr, addrLen)
	if errno != 0 {
		return errno
	}

	logf("connect %d %s", fd, addr)

	switch {
	case addr.Addr().Is4():
	default:
		return syscall.EINVAL
	}

	stream, err := l.machine.netstack.OpenStream(addr)
	if err != nil {
		return syscall.EINVAL
	}

	// XXX
	sock.Stream = stream

	// XXX: non blocking now
	return syscall.EINPROGRESS
}

func (l *LinuxOS) SysGetrandom(ptr syscallabi.ByteSliceView, flags int) (int, error) {
	// TODO: implement entirely in userspace?

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		return 0, syscall.EINVAL
	}

	logf("getrandom %d %d", ptr.Len(), flags)

	if flags != 0 {
		return 0, syscall.EINVAL
	}

	n := 0
	for ptr.Len() > 0 {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], gosimruntime.Fastrand64())
		m := ptr.Write(buf[:])
		ptr = ptr.SliceFrom(m)
		n += m
	}
	return n, nil
}

func (l *LinuxOS) SysGetpid() int {
	// TODO: return some random number instead?
	return 42
}

func (l *LinuxOS) SysUname(buf syscallabi.ValueView[Utsname]) (err error) {
	ptr := (*Utsname)(buf.UnsafePointer())
	name := syscallabi.NewSliceView(&ptr.Nodename[0], uintptr(len(ptr.Nodename)))
	var nameArray [65]int8
	n := min(len(l.machine.label), 64)
	for i := 0; i < n; i++ {
		nameArray[i] = int8(l.machine.label[i])
	}
	nameArray[n] = 0
	name.Write(nameArray[:n+1])
	return nil
}
