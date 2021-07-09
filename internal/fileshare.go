package internal

import "syscall"

// PendingFileshare is a fileshare that is yet to be downloaded.
// This is created when a user updates a file, and deleted once another user downloaded it.
type PendingFileshare struct {
	RawConn  syscall.RawConn
	FileSize int
	FileName string
}
