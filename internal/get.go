package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"net/http"
	"syscall"
)

type GetOperation struct {
	repository *PendingFileshareRepository
}

func NewGetOperation(repository *PendingFileshareRepository) GetOperation {
	return GetOperation{
		repository: repository,
	}
}

func (o *GetOperation) Get(ctx context.Context, w http.ResponseWriter, resourceName string) error {
	log.Printf("Get for %q", resourceName)

	fileshare, ok := o.repository.GetAndDelete(resourceName)
	if !ok {
		return &BadRequestError{Err: fmt.Errorf("resource %q is not found", resourceName)}
	}

	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 200 OK\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", fileshare.FileName))
	buf.WriteString("Content-Type: application/octet-stream\r\n")
	buf.WriteString("\r\n")

	readPipe, writePipe, err := openPipes()
	if err != nil {
		return fmt.Errorf("could not open pipe: %w", err)
	}
	defer cleanupPipe(readPipe, writePipe)

	downloaderConn, err := hijackConnection(w)
	if err != nil {
		return err
	}

	downloaderRawConn, err := extractRawConn(downloaderConn)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		bytesLeft := fileshare.FileSize
		return doSyscallOperation(fileshare.RawConn.Read, func(uploaderFd int) error {
			defer closeFd(uploaderFd)
			for bytesLeft > 0 {
				n, err := splice(uploaderFd, writePipe, bytesLeft)
				if err != nil {
					return fmt.Errorf("read splice error: %w", err)
				}
				bytesLeft -= int(n)
			}

			var buf bytes.Buffer
			buf.WriteString("HTTP/1.1 204 No Content\r\n")
			buf.WriteString("\r\n")
			_, err := syscall.Write(uploaderFd, buf.Bytes())
			if err != nil {
				return fmt.Errorf("could not write: %w", err)
			}
			return nil
		})
	})

	g.Go(func() error {
		bytesLeft := fileshare.FileSize
		return doSyscallOperation(downloaderRawConn.Write, func(downloaderFd int) error {
			defer closeFd(downloaderFd)
			_, err := syscall.Write(downloaderFd, buf.Bytes())
			if err != nil {
				return fmt.Errorf("could not write: %w", err)
			}

			for bytesLeft > 0 {
				n, err := splice(readPipe, downloaderFd, bytesLeft)
				if err != nil {
					return fmt.Errorf("write splice error: %w", err)
				}
				bytesLeft -= int(n)
			}
			return nil
		})
	})
	if err := g.Wait(); err != nil {
		log.Printf("error while transfering: %w", err)
	}
	// We can't return any error the the user because we already started transferring things on the connection.
	return nil

}

func hijackConnection(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("responseWrite does not support hijack")
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("error while hijacking the connection: %w", err)
	}
	return conn, nil
}

func extractRawConn(conn net.Conn) (syscall.RawConn, error) {
	syscallConn, ok := conn.(syscall.Conn)
	if !ok {
		return nil, errors.New("cannot extract syscall connection")
	}

	rawConn, err := syscallConn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("could not get syscall connection: %w", err)
	}

	return rawConn, nil
}

func openPipes() (int, int, error) {
	var pipefd [2]int
	err := syscall.Pipe(pipefd[:])
	if err != nil {
		return 0, 0, fmt.Errorf("could not open pipe: %w", err)
	}
	return pipefd[0], pipefd[1], nil
}

func splice(from, to, length int) (int64, error) {
	n, err := syscall.Splice(from, nil, to, nil, length, unix.SPLICE_F_MOVE)
	if err != nil {
		details := ""
		switch err {
		case syscall.EAGAIN:
			details = "Resource temporarily unavailable"
		case syscall.EINVAL:
			details = "Invalid argument"
		case syscall.ENOENT:
			details = "No such file or directory"
		}
		return n, fmt.Errorf("Could not write %q: %w", details, err)
	}

	return n, nil

}

type syscallOperation = func(func(fd uintptr) bool) error

func doSyscallOperation(method syscallOperation, f func(fd int) error) error {
	errCh := make(chan error)
	err := method(func(fd uintptr) bool {
		callbackErr := f(int(fd))
		errCh <- callbackErr
		return true
	})
	if err != nil {
		return err
	}
	if <-errCh != nil {
		return err
	}
	return nil
}

func cleanupPipe(readPipe, writePipe int) {
	if err := syscall.Close(writePipe); err != nil {
		log.Printf("Could not close write pipe: %w", err)
	}
	log.Printf("cleaned write pipe")

	if err := syscall.Close(readPipe); err != nil {
		log.Printf("Could not close read pipe: %w", err)
	}
	log.Printf("cleaned read pipe")
}

func closeFd(fd int) {
	if err := syscall.Close(fd); err != nil {
		log.Printf("could not close connection: %w", err)
	}
}
