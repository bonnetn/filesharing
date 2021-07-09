package internal

import (
	"sync"
)

type PendingFileshareRepository interface {
	PendingFileshareGetter
	PendingFileshareSetter
}

type PendingFileshareGetter interface {
	GetAndDelete(key string) (PendingFileshare, bool)
}

type PendingFileshareSetter interface {
	Set(string, PendingFileshare) bool
}

func NewPendingFileshareRepository() PendingFileshareRepository {
	return &inMemoryFileshareRepository{}
}

type inMemoryFileshareRepository struct {
	sync.Mutex

	pendingFileshares map[string]PendingFileshare
}

// GetAndDelete retrieves a PendingFileshare from the repository and then deletes it.
func (r *inMemoryFileshareRepository) GetAndDelete(key string) (PendingFileshare, bool) {
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
func (r *inMemoryFileshareRepository) Set(key string, fd PendingFileshare) bool {
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
