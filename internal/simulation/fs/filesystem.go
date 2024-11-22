package fs

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"
	"syscall"

	"github.com/jellevandenhooff/gosim/internal/simulation/syscallabi"
)

// next:
// - delete on test end, somehow (but pool isn't shared between tests? maybe we could/should make it?)
// - think about where we should invoke GC and why that's safe
// -- eg. when we create/apply multiple ops. should there be an API for doing that atomically?

// after a crash, iterate over all files and check for zeroes and if so, delete (optimize later: track orphans?)

// XXX: handle bad requests somehow?

type backingFile struct {
	inode int
	file  *chunkedFile

	linkCount int
}

var zeroes = make([]byte, 1024)

func (f *backingFile) read(pos int64, buffer syscallabi.ByteSliceView) int {
	if int(pos) < f.file.size {
		n := min(buffer.Len(), f.file.size-int(pos))
		f.file.Read(int(pos), buffer.Slice(0, n))
		return n
	}
	return 0
}

func (f *backingFile) len() int {
	return f.file.size
}

func (f *backingFile) resize(size int) {
	f.file.Resize(size)
}

func (f *backingFile) write(pos int, size int, chunks []*refCountedChunk) {
	f.file.Write(pos, size, chunks)
}

func (f *backingFile) clone() *backingFile {
	return &backingFile{
		inode:     f.inode,
		file:      f.file.Clone(),
		linkCount: f.linkCount, // XXX: test that link count is maintained
	}
}

type backingDir struct {
	inode   int
	entries map[string]int
}

const rootInode = 1

type filesystemState struct {
	objects map[int]any
	next    int
}

func (fss *filesystemState) clone() *filesystemState {
	objects := make(map[int]any)
	for inode, obj := range fss.objects {
		switch obj := obj.(type) {
		case *backingFile:
			objects[inode] = obj.clone()
		case *backingDir:
			objects[inode] = &backingDir{
				inode:   obj.inode,
				entries: maps.Clone(obj.entries),
			}
		}
	}

	return &filesystemState{
		objects: objects,
		next:    fss.next,
	}
}

type Filesystem struct {
	mu sync.Mutex

	// for now: flat filesystem, only one directory

	pendingOps *pendingOps

	// XXX: jank...
	openCountByInode    map[int]int
	pendingCountByInode map[int]int

	// XXX: make a different, higher-level type?
	mem       *filesystemState
	persisted *filesystemState // XXX: rename to "disk"?
}

func (fs *Filesystem) apply(op fsOp) {
	/*
		switch op := op.(type) {
		case *resizeOp:
			slog.Info("resize", "size", op.size)
		case *writeOp:
			slog.Info("write", "size", len(op.data), "pos", op.pos)
		}
	*/
	fs.pendingOps.addOp(op)
	op.touchedInodesIter(func(inode int) bool {
		fs.pendingCountByInode[inode]++
		return true
	})
	op.apply(fs.mem, fs)
}

func (fs *Filesystem) releaseOpRefs(ops []*pendingOp) {
	for _, op := range ops {
		op.fsOp.touchedInodesIter(func(inode int) bool {
			newCount := fs.pendingCountByInode[inode] - 1
			if newCount == 0 {
				delete(fs.pendingCountByInode, inode)
				// XXX: this check here does double work. any file whose links
				// hit zero MUST eventually be flushed by an op that refers to
				// it, which will be this op, and so here we can delete it.
				fs.maybeGC(inode)
			} else {
				fs.pendingCountByInode[inode] = newCount
			}
			return true
		})
	}
}

func (fs *Filesystem) flushInode(inode int) {
	ops := fs.pendingOps.flushInode(inode, fs.pendingOps.buffer)
	for _, op := range ops {
		op.fsOp.apply(fs.persisted, fs)
		// XXX: janky but works for now. release chunks.
		if op, ok := op.fsOp.(*writeOp); ok {
			for i, chunk := range op.chunks {
				chunk.decRef()
				op.chunks[i] = nil // set to nil just in case?
			}
			op.chunks = nil // set to nil just in case?
		}
	}

	fs.releaseOpRefs(ops)
	fs.pendingOps.releaseBuffer(ops)
}

type gcer interface {
	maybeGC(inode int)
}

type fsOp interface {
	apply(fs *filesystemState, gcer gcer)
	addDeps(p *pendingOps, op *pendingOp)
	touchedInodesIter(func(inode int) bool)
	String() string
}

func inititalState() *filesystemState {
	return &filesystemState{
		objects: map[int]any{
			rootInode: &backingDir{
				inode:   rootInode,
				entries: make(map[string]int),
			},
		},
		next: rootInode + 1,
	}
}

func NewFilesystem() *Filesystem {
	return &Filesystem{
		pendingOps:          newPendingOps(),
		mem:                 inititalState(),
		persisted:           inititalState(),
		openCountByInode:    make(map[int]int),
		pendingCountByInode: make(map[int]int),
	}
}

func (fs *filesystemState) getFile(idx int) *backingFile {
	return fs.objects[idx].(*backingFile)
}

func (fs *filesystemState) getDir(idx int) *backingDir {
	return fs.objects[idx].(*backingDir)
}

type writeOp struct {
	// XXX: should this be an inode number?
	inode int

	pos    int
	size   int
	chunks []*refCountedChunk
}

func (o *writeOp) apply(fs *filesystemState, gcer gcer) {
	file := fs.getFile(o.inode)
	file.write(o.pos, o.size, o.chunks)
}

func (o *writeOp) touchedInodesIter(iter func(inode int) bool) {
	iter(o.inode)
}

func (o *writeOp) String() string {
	return fmt.Sprintf("write file:%d pos:%d size:%d", o.inode, o.pos, o.size)
}

type resizeOp struct {
	inode int
	size  int
}

func (o *resizeOp) apply(fs *filesystemState, gcer gcer) {
	file := fs.getFile(o.inode)
	file.resize(o.size)
}

func (o *resizeOp) touchedInodesIter(iter func(inode int) bool) {
	iter(o.inode)
}

func (o *resizeOp) String() string {
	return fmt.Sprintf("resize file:%d size:%d", o.inode, o.size)
}

type dirOpPart struct {
	dir  int
	name string
	file int
}

func (p *dirOpPart) String() string {
	return fmt.Sprintf("dir:%d name:%s file:%d", p.dir, p.name, p.file)
}

type dirOp struct {
	parts []dirOpPart
}

func (o *dirOp) apply(fs *filesystemState, gcer gcer) {
	for _, p := range o.parts {
		dir := fs.getDir(p.dir)

		old := dir.entries[p.name]
		if file, ok := fs.objects[old].(*backingFile); ok {
			file.linkCount--
			// slog.Info("fs link count", "inode", file.inode, "count", file.linkCount)
			if file.linkCount == 0 {
				gcer.maybeGC(old) // XXX: jank. think about locking/ordering/help
			}
		}

		if p.file == -1 {
			delete(dir.entries, p.name)
		} else {
			dir.entries[p.name] = p.file
			if file, ok := fs.objects[p.file].(*backingFile); ok {
				file.linkCount++
				// slog.Info("fs link count", "inode", file.inode, "count", file.linkCount)
			}
		}
	}
}

func (o *dirOp) touchedInodesIter(iter func(inode int) bool) {
	for _, p := range o.parts {
		if !iter(p.file) || !iter(p.dir) {
			return
		}
	}
}

func (o *dirOp) String() string {
	return fmt.Sprintf("dirop %v", o.parts)
}

type allocOp struct {
	dir bool

	inode int
}

func (o *allocOp) apply(fs *filesystemState, gcer gcer) {
	if _, ok := fs.objects[o.inode]; ok {
		panic("double alloc")
	}
	if o.dir {
		fs.objects[o.inode] = &backingDir{
			inode:   o.inode,
			entries: make(map[string]int),
		}
	} else {
		fs.objects[o.inode] = &backingFile{
			inode: o.inode,
			file:  &chunkedFile{},
		}
	}
	if o.inode >= fs.next {
		fs.next = o.inode + 1
	}
}

func (o *allocOp) touchedInodesIter(iter func(inode int) bool) {
	iter(o.inode)
}

func (o *allocOp) String() string {
	return fmt.Sprintf("alloc inode:%d dir:%v", o.inode, o.dir)
}

func (o *allocOp) addDeps(p *pendingOps, op *pendingOp) {
	p.addDep(op, depKey{kind: depKeyAlloc, inode: int32(o.inode)}, depStoreAs)
	p.addDep(op, depKey{kind: depKeyResizeFile, inode: int32(o.inode)}, depStoreAs) // optimization?
}

func (o *writeOp) addDeps(p *pendingOps, op *pendingOp) {
	p.addDep(op, depKey{kind: depKeyWriteFile, inode: int32(o.inode)}, depStoreAs)
	// p.addDep(op, depKey{kind: depKeyAlloc, inode: int32(o.inode)}, depDependsOn) // optimization?
	p.addDep(op, depKey{kind: depKeyResizeFile, inode: int32(o.inode)}, depDependsOn)
}

func (o *resizeOp) addDeps(p *pendingOps, op *pendingOp) {
	// p.addDep(op, depKey{kind: depKeyAlloc, inode: int32(o.inode)}, depDependsOn) // optimization?
	p.addDep(op, depKey{kind: depKeyResizeFile, inode: int32(o.inode)}, depStoreAsAndDependsOn)
}

func (o *dirOp) addDeps(p *pendingOps, op *pendingOp) {
	for _, part := range o.parts {
		p.addDep(op, depKey{kind: depKeyAlloc, inode: int32(part.file)}, depDependsOn)
		p.addDep(op, depKey{kind: depKeyAlloc, inode: int32(part.dir)}, depDependsOn)
		p.addDep(op, depKey{kind: depKeyWriteDir, inode: int32(part.dir)}, depStoreAs)
		p.addDep(op, depKey{kind: depKeyWriteDirEntry, inode: int32(part.dir), entry: part.name}, depStoreAsAndDependsOn)
	}
}

type nilgc struct{}

func (nilgc) maybeGC(inode int) {}

type crashIterator struct {
	workCh chan bool
	nextCh chan []fsOp
}

func (c *crashIterator) run(ops *pendingOps) {
	c.workCh = make(chan bool)
	c.nextCh = make(chan []fsOp)
	go func() {
		if !<-c.workCh {
			return
		}
		ops.iterAllCrashes(func(ops []fsOp) bool {
			c.nextCh <- ops
			return <-c.workCh
		})
		close(c.nextCh)
	}()
}

func (c *crashIterator) next() ([]fsOp, bool) {
	c.workCh <- true
	next, ok := <-c.nextCh
	return next, ok
}

/*
func (c *crashIterator) stop() {
	c.workCh <- false
	<-c.nextCh
}
*/

func (fs *Filesystem) Release() {
	// XXX: somehow free resources here.
}

func (fs *Filesystem) FlushEverything() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// XXX: these methods for handling stop/crash/restart are kinda janky...
	// how do we free things? how do we not needlessly copy (eg. after a restart
	// where we don't care about the old? etc.)

	fs.persisted = fs.mem.clone()
	fs.pendingOps = newPendingOps()
	fs.pendingCountByInode = make(map[int]int)
}

func (fs *Filesystem) CrashClone(partialDisk bool) *Filesystem {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// XXX: think about if we do the right thing with the next inode number

	persisted := fs.persisted.clone()
	if partialDisk {
		pickedOps := fs.pendingOps.randomCrashSubset()
		for _, op := range pickedOps {
			op.apply(persisted, nilgc{}) // XXX: jank think about gc
		}
	}
	mem := persisted.clone()

	return &Filesystem{
		persisted:           persisted,
		mem:                 mem,
		pendingOps:          newPendingOps(),
		openCountByInode:    make(map[int]int),
		pendingCountByInode: make(map[int]int),
	}
}

func (fs *Filesystem) IterCrashes() func() (*Filesystem, bool) {
	var iter crashIterator
	iter.run(fs.pendingOps)

	// XXX: handle leaking??

	return func() (*Filesystem, bool) {
		ops, ok := iter.next()
		if !ok {
			return nil, false
		}

		newPersisted := fs.persisted.clone()
		for _, op := range ops {
			op.apply(newPersisted, nilgc{}) // XXX: jank think about gc
		}
		// XXX: find zero-ref files and delete them
		return &Filesystem{
			persisted:           newPersisted,
			mem:                 newPersisted.clone(),
			pendingOps:          newPendingOps(),
			openCountByInode:    make(map[int]int),
			pendingCountByInode: make(map[int]int),
		}, true
	}
}

/*
func simCrash(fs *filesystem) {
	// XXX: return a new *filesystem instead?
	// not entirely necessary in this case and maybe less efficient since we'd have to clone persisted

	ops := fs.pendingOps.randomCrashSubset()
	for _, op := range ops {
		op.apply(fs.persisted)
	}

	fs.mem = fs.persisted.clone()
	fs.pendingOps = newPendingOps()
}
*/

func tryGetLinkCount(s *filesystemState, inode int) (int, bool) {
	switch file := s.objects[inode].(type) {
	case *backingFile:
		return file.linkCount, true
	}
	return 0, false
}

type InodeInfo struct {
	MemLinks   int
	MemExists  bool
	DiskLinks  int
	DiskExists bool
	Handles    int
	Ops        int
}

func (fs *Filesystem) getInodeInfoLocked(inode int) InodeInfo {
	mem, memOk := tryGetLinkCount(fs.mem, inode)
	disk, diskOk := tryGetLinkCount(fs.persisted, inode)
	return InodeInfo{
		// XXX: also return if exists or not
		Handles:    fs.openCountByInode[inode],
		Ops:        fs.pendingCountByInode[inode],
		MemLinks:   mem,
		MemExists:  memOk,
		DiskLinks:  disk,
		DiskExists: diskOk,
	}
}

func (fs *Filesystem) maybeGC(inode int) {
	if inode == 2 {
		// slog.Info("considering gc", "inode", inode, "info", fs.getInodeInfoLocked(inode))
	}

	// haha take taht!!!
	if _, ok := fs.openCountByInode[inode]; ok {
		return
	}
	if _, ok := fs.pendingCountByInode[inode]; ok {
		return
	}
	switch file := fs.mem.objects[inode].(type) {
	case *backingFile:
		if file.linkCount == 0 {
			// XXX: release the file here
			file.file.Free()
			file.file = nil
			delete(fs.mem.objects, inode)
			// delete? but what about persisted????????????????????????????????????? (must be zero also or we are in a strange situation)
			// slog.Info("fs freeing mem", "inode", inode)
		}
	}
	switch file := fs.persisted.objects[inode].(type) {
	case *backingFile:
		if file.linkCount == 0 {
			// XXX: release the file here
			file.file.Free()
			file.file = nil
			delete(fs.persisted.objects, inode)
			// delete? but what about persisted????????????????????????????????????? (must be zero also or we are in a strange situation)
			// slog.Info("fs freeing persisted", "inode", inode)
		}
	}
}

func (fs *Filesystem) Read(inode int, pos int64, buffer syscallabi.ByteSliceView) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file := fs.mem.getFile(inode)
	return file.read(pos, buffer)
}

func (fs *Filesystem) Write(inode int, pos int64, buffer syscallabi.ByteSliceView) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file := fs.mem.getFile(inode)

	if int(pos)+buffer.Len() > file.len() {
		op := &resizeOp{
			inode: inode,
			size:  int(pos) + buffer.Len(),
		}
		fs.apply(op)
	}

	chunks := makeChunks(int(pos), buffer)
	len := buffer.Len()

	op := &writeOp{
		inode:  inode,
		pos:    int(pos),
		size:   buffer.Len(),
		chunks: chunks,
		// XXX: if we're running in some kind of non-fail mode, no need to make an (expensive?) copy?
		// data: data, // XXX: test that we clone this please somehow (also other clones)
	}
	fs.apply(op)

	return len
}

func (fs *Filesystem) Truncate(inode, pos int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file := fs.mem.getFile(inode)

	if pos != file.len() {
		op := &resizeOp{
			inode: inode,
			size:  pos,
		}
		fs.apply(op)
	}
}

func (fs *Filesystem) Sync(inode int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.flushInode(inode)
}

func (fs *Filesystem) Mkdir(name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.mem.getDir(rootInode).entries[name]; ok {
		return errors.New("help already exists")
	}

	inode := fs.mem.next
	op := &allocOp{
		dir:   true,
		inode: inode,
	}
	fs.apply(op)

	dOp := &dirOp{
		parts: []dirOpPart{
			{
				dir:  rootInode,
				name: name,
				file: inode,
			},
		},
	}
	fs.apply(dOp)

	return nil
}

type StatResp struct {
	Inode int
	IsDir bool
}

// XXX: these two stats should share implementation

func (fs *Filesystem) statLocked(inode int) (StatResp, error) {
	obj := fs.mem.objects[inode]
	switch obj.(type) {
	case *backingDir:
		return StatResp{
			Inode: inode,
			IsDir: true,
		}, nil

	case *backingFile:
		return StatResp{
			Inode: inode,
			IsDir: false,
		}, nil

	default:
		panic("invalid")
	}
}

func (fs *Filesystem) Stat(path string) (StatResp, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.mem.getDir(rootInode)
	inode, ok := dir.entries[path]
	if !ok {
		return StatResp{}, syscall.ENOENT
	}
	return fs.statLocked(inode)
}

func (fs *Filesystem) Statfd(inode int) (StatResp, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return fs.statLocked(inode)
}

func (fs *Filesystem) Remove(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.mem.getDir(rootInode)
	if _, ok := dir.entries[path]; !ok {
		return syscall.ENOENT
	}

	dOp := &dirOp{
		parts: []dirOpPart{
			{
				dir:  rootInode,
				name: path,
				file: -1,
			},
		},
	}
	// XXX: clean up backing?
	fs.apply(dOp)

	return nil
}

func (fs *Filesystem) Rename(from, to string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.mem.getDir(rootInode)
	inode, ok := dir.entries[from]
	if !ok {
		return syscall.ENOENT
	}

	// slog.Info("fs rename", "from", from, "to", to, "inode", inode)

	dOp := &dirOp{
		parts: []dirOpPart{
			{
				dir:  rootInode,
				name: from,
				file: -1,
			},
			{
				dir:  rootInode,
				name: to,
				file: inode,
			},
		},
	}
	fs.apply(dOp)

	return nil
}

func (fs *Filesystem) OpenFile(path string, flag int) (int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	inode, ok := fs.mem.getDir(rootInode).entries[path]
	if path == "." {
		// XXX: JANK
		inode = rootInode
		ok = true
	}
	if !ok {
		if flag&os.O_CREATE == 0 {
			return 0, syscall.ENOENT
		}

		inode = fs.mem.next
		op := &allocOp{
			dir:   false,
			inode: inode,
		}
		fs.apply(op)

		dOp := &dirOp{
			parts: []dirOpPart{
				{
					dir:  rootInode,
					name: path,
					file: inode,
				},
			},
		}
		fs.apply(dOp)

		// slog.Info("fs create", "path", args.Path, "inode", inode)
	}
	if ok && flag&os.O_EXCL != 0 {
		return 0, errors.New("file already existed")
	}
	if ok && flag&os.O_TRUNC != 0 {
		file := fs.mem.getFile(inode)
		if file.len() != 0 {
			op := &resizeOp{
				inode: inode,
				size:  0,
			}
			fs.apply(op)
		}
	}
	fs.openCountByInode[inode]++

	return inode, nil
}

func (fs *Filesystem) CloseFile(inode int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// XXX: make this API handle based so it's less brittle?
	newCount := fs.openCountByInode[inode] - 1
	if newCount == 0 {
		delete(fs.openCountByInode, inode)
		fs.maybeGC(inode)
	} else {
		fs.openCountByInode[inode] = newCount
	}

	return nil
}

type ReadDirEntry struct {
	Name  string
	IsDir bool
}

func (fs *Filesystem) ReadDir(name string) ([]ReadDirEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// XXX: jank
	if name != "." {
		return nil, errors.New("bad dir")
	}
	dir := fs.mem.getDir(rootInode)

	var entries []ReadDirEntry
	for name, inode := range dir.entries {
		var de ReadDirEntry

		obj := fs.mem.objects[inode]
		switch obj.(type) {
		case *backingDir:
			de = ReadDirEntry{
				Name:  name, // XXX?
				IsDir: true,
			}

		case *backingFile:
			de = ReadDirEntry{
				Name:  name, // XXX?
				IsDir: false,
			}

		default:
			panic("invalid")
		}

		entries = append(entries, de)
	}
	slices.SortFunc(entries, func(a, b ReadDirEntry) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return entries, nil
}

func (fs *Filesystem) GetInodeInfo(inode int) InodeInfo {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.getInodeInfoLocked(inode)
}
