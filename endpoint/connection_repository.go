package endpoint

import (
	"sync"
	"syscall"
)

type Channel struct {
	RawConn  syscall.RawConn
	FileSize int
	FileName string
}

type ChannelRepository struct {
	sync.Mutex

	channels map[string]Channel
}

// GetAndDelete retrieves a Channel from the repository and then deletes it.
func (r *ChannelRepository) GetAndDelete(key string) (Channel, bool) {
	r.Lock()
	defer r.Unlock()

	channel, ok := r.channels[key]
	if !ok {
		return Channel{}, false
	}

	delete(r.channels, key)
	return channel, true
}

// Set adds a new Channel in the repository.
// If the key already exists, it returns false, otherwise it returns true.
func (r *ChannelRepository) Set(key string, fd Channel) bool {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.channels[key]; ok {
		return false // Key already exists.
	}

	if r.channels == nil {
		r.channels = make(map[string]Channel)
	}

	r.channels[key] = fd
	return true
}
