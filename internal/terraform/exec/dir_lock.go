package exec

import "sync"

// DirLock serializes terraform CLI operations per working directory.
// Multiple ExecService instances (created via WithDir) share a single
// DirLock to prevent concurrent terraform processes in the same directory.
type DirLock struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewDirLock() *DirLock {
	return &DirLock{locks: make(map[string]*sync.Mutex)}
}

func (d *DirLock) Acquire(dir string) { d.forDir(dir).Lock() }
func (d *DirLock) Release(dir string) { d.forDir(dir).Unlock() }

func (d *DirLock) forDir(dir string) *sync.Mutex {
	d.mu.Lock()
	defer d.mu.Unlock()
	if m, ok := d.locks[dir]; ok {
		return m
	}
	m := &sync.Mutex{}
	d.locks[dir] = m
	return m
}
