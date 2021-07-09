package internal

import "syscall"

type PendingFileshare struct {
	RawConn  syscall.RawConn
	FileSize int
	FileName string
}
