package internal

import (
	"bufio"
	"net"
)

// PendingFileshare is a fileshare that is yet to be downloaded.
// This is created when a user updates a file, and deleted once another user downloaded it.
type PendingFileshare struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	Writer   *bufio.Writer
	FileSize int64
	FileName string
}
