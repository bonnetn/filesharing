package internal

import (
	"sync"
	"syscall"
)

type PendingFileshare struct {
	RawConn  syscall.RawConn
	FileSize int
	FileName string
}

type PendingFileshareRepository struct {
	sync.Mutex

	pendingFileshares map[string]PendingFileshare
}

func NewPendingFileshareRepository() PendingFileshareRepository {
	return PendingFileshareRepository{}
}

// GetAndDelete retrieves a PendingFileshare from the repository and then deletes it.
func (r *PendingFileshareRepository) GetAndDelete(key string) (PendingFileshare, bool) {
	r.Lock()
	defer r.Unlock()

	fileshare, ok := r.pendingFileshares[key]
	if !ok {
		return PendingFileshare{}, false
	}

	delete(r.pendingFileshares, key)
	return fileshare, true
}

// Set adds a new PendingFileshare in the repository.
// If the key already exists, it returns false, otherwise it returns true.
func (r *PendingFileshareRepository) Set(key string, fd PendingFileshare) bool {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.pendingFileshares[key]; ok {
		return false // Key already exists.
	}

	if r.pendingFileshares == nil {
		r.pendingFileshares = make(map[string]PendingFileshare)
	}

	r.pendingFileshares[key] = fd
	return true
}
