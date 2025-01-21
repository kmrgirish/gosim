package go123

import (
	"github.com/kmrgirish/gosim/gosimruntime"
)

func Sync_runtime_Semacquire(addr *uint32) {
	gosimruntime.Semacquire(addr, false)
}

func Sync_runtime_Semrelease(addr *uint32, handoff bool, skipframes int) {
	gosimruntime.Semrelease(addr)
}

func Sync_runtime_SemacquireMutex(addr *uint32, lifo bool, skipframes int) {
	gosimruntime.Semacquire(addr, lifo)
}

func Sync_runtime_SemacquireRWMutexR(addr *uint32, lifo bool, skipframes int) {
	gosimruntime.Semacquire(addr, lifo)
}

func Sync_runtime_SemacquireRWMutex(addr *uint32, lifo bool, skipframes int) {
	gosimruntime.Semacquire(addr, lifo)
}

func Sync_runtime_registerPoolCleanup(cleanup func()) {
	// noop
}

func Sync_runtime_procPin() int {
	// noop
	// TODO: randomize?
	return 0
}

func Sync_runtime_procUnpin() {
	// noop
}

func Sync_runtime_nanotime() int64 {
	return gosimruntime.Nanotime()
}

// Active spinning runtime support.
// runtime_canSpin reports whether spinning makes sense at the moment.
func Sync_runtime_canSpin(i int) bool {
	return false
}

// runtime_doSpin does active spinning.
func Sync_runtime_doSpin() {
	panic("no")
}

//go:norace
func Sync_runtime_LoadAcquintptr(ptr *uintptr) uintptr {
	// XXX
	if gosimruntime.AtomicYield {
		gosimruntime.Yield()
	}
	return *ptr
}

//go:norace
func Sync_runtime_StoreReluintptr(ptr *uintptr, val uintptr) uintptr {
	if gosimruntime.AtomicYield {
		gosimruntime.Yield()
	}
	*ptr = val
	return 0 // XXX?
}

type NotifyList struct {
	inner gosimruntime.NotifyList
}

func Sync_runtime_notifyListAdd(l *NotifyList) uint32 {
	return l.inner.Add()
}

func Sync_runtime_notifyListWait(l *NotifyList, t uint32) {
	l.inner.Wait(t)
}

func Sync_runtime_notifyListNotifyAll(l *NotifyList) {
	l.inner.NotifyAll()
}

func Sync_runtime_notifyListNotifyOne(l *NotifyList) {
	l.inner.NotifyOne()
}

// Ensure that sync and runtime agree on size of notifyList.
func Sync_runtime_notifyListCheck(size uintptr) {
	// noop
}

func Sync_runtime_randn(n uint32) uint32 {
	return gosimruntime.Fastrandn(n)
}

func Sync_throw(str string) {
	panic(str)
}

func Sync_fatal(str string) {
	panic(str)
}
