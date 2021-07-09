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

type FileshareGetter interface {
	Get(ctx context.Context, w http.ResponseWriter, resourceName string) error
}

func NewGetOperation(repository PendingFileshareGetter) FileshareGetter {
	return &get{
		repository: repository,
	}
}

type get struct {
	repository PendingFileshareGetter
}

// Get retrieves a pending fileshare and send the response to the caller.
func (o *get) Get(ctx context.Context, w http.ResponseWriter, resourceName string) error {
	log.Printf("Get for %q", resourceName)

	fileshare, ok := o.repository.GetAndDelete(resourceName)
	if !ok {
		return &BadRequestError{Err: fmt.Errorf("resource %q is not found", resourceName)}
	}

	readPipe, writePipe, err := createPipe()
	if err != nil {
		return fmt.Errorf("could not open pipe: %w", err)
	}

	downloaderConn, err := hijackConnection(w)
	if err != nil {
		return err
	}

	if err := o.get(ctx, downloaderConn, fileshare, readPipe, writePipe); err != nil {
		// NOTE: We cannot return an error to the user here since we hijacked the connection.
		log.Printf("get failed: %v", err)
	}
	return nil
}

func (o *get) get(ctx context.Context, downloaderConn net.Conn, fileshare PendingFileshare, readPipe, writePipe int) error {
	downloaderRawConn, err := extractRawConn(downloaderConn)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		bytesLeft := fileshare.FileSize
		return doSyscallOperation(fileshare.RawConn.Read, func(uploaderFd int) error {
			for bytesLeft > 0 {
				n, err := splice(uploaderFd, writePipe, bytesLeft)
				if err != nil {
					return fmt.Errorf("read splice error: %w", err)
				}
				bytesLeft -= int(n)
			}

			_, err := syscall.Write(uploaderFd, httpPayloadForSuccessfulUpload())
			if err != nil {
				return fmt.Errorf("could not write: %w", err)
			}

			closeFd(uploaderFd)
			closeFd(writePipe)
			return nil
		})
	})

	g.Go(func() error {
		bytesLeft := fileshare.FileSize
		return doSyscallOperation(downloaderRawConn.Write, func(downloaderFd int) error {
			_, err := syscall.Write(downloaderFd, httpPreludeForFileDownload(fileshare.FileName))
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
			closeFd(readPipe)
			closeFd(downloaderFd)
			return nil
		})
	})
	return g.Wait()

}

// hijackConnection takes ownership of underlying net.Conn.
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

// extractRawConn extracts the syscall.RawConn from a net.Conn.
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

// createPipe creates a pipe (Unix).
func createPipe() (int, int, error) {
	var pipefd [2]int
	err := syscall.Pipe(pipefd[:])
	if err != nil {
		return 0, 0, fmt.Errorf("could not open pipe: %w", err)
	}
	return pipefd[0], pipefd[1], nil
}

// splice calls the splice syscall.
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

func httpPreludeForFileDownload(filename string) []byte {
	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 200 OK\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", filename))
	buf.WriteString("Content-Type: application/octet-stream\r\n")
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func httpPayloadForSuccessfulUpload() []byte {
	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 204 No Content\r\n")
	buf.WriteString("\r\n")
	return buf.Bytes()
}

