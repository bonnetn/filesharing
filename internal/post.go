package internal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"syscall"
)

const (
	fileSizeHeader   = "x-filesharing-file-size"
	formDataPartName = "file_to_upload"
)

type FileshareCreator interface {
	Create(ctx context.Context, w http.ResponseWriter, resourceName string, r *http.Request) error
}

func NewCreateOperation(repository PendingFileshareSetter) FileshareCreator {
	return &create{
		repository: repository,
	}
}

type create struct {
	repository PendingFileshareSetter
}

// Create makes a new pending fileshare.
func (o *create) Create(ctx context.Context, w http.ResponseWriter, resourceName string, r *http.Request) error {
	log.Printf("Create %q", resourceName)

	fileSize, err := extractFileSize(r.Header)
	if err != nil {
		return &BadRequestError{Err: err}
	}

	boundary, err := extractFormDataBoundary(r.Header)
	if err != nil {
		return &BadRequestError{Err: err}
	}

	conn, err := hijackConnection(w)
	if err != nil {
		return err
	}

	if err := o.create(ctx, conn, resourceName, fileSize, boundary); err != nil {
		// NOTE: We cannot return an error to the user here since we hijacked the connection.
		log.Printf("create failed: %v", err)
	}

	return nil
}

func (o *create) create(ctx context.Context, conn net.Conn, resourceName string, fileSize uint64, boundary string) error {
	rawConn, err := extractRawConn(conn)
	if err != nil {
		return err
	}

	var (
		readErr  error
		filename string
	)
	err = rawConn.Read(func(fd uintptr) (done bool) {
		if readErr = skipBoundary(int(fd), boundary); readErr != nil {
			return true
		}

		filename, readErr = skipFormDataHeaders(int(fd))
		if readErr != nil {
			return true
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("could not read from raw connection: %w", err)
	}
	if readErr != nil {
		return &BadRequestError{Err: readErr}
	}

	c := PendingFileshare{
		RawConn:  rawConn,
		FileSize: int(fileSize),
		FileName: filename,
	}
	if !o.repository.Set(resourceName, c) {
		return &BadRequestError{Err: fmt.Errorf("there is already a file waiting to be downloaded for %q", resourceName)}
	}
	return nil
}

// skipBoundary skips the first line of formdata.
func skipBoundary(fd int, boundary string) error {
	firstLine, err := readLine(fd)
	if err != nil {
		return fmt.Errorf("couldn't read line: %w", err)
	}

	if strings.Compare(string(firstLine), "--"+boundary) != 0 {
		return errors.New("wrong boundary")
	}

	return nil
}

// skipFormDataHeaders skips the formdata headers.
func skipFormDataHeaders(fd int) (string, error) {
	var rawPartHeaders bytes.Buffer
	for {
		line, err := readLine(fd)
		if err != nil {
			return "", fmt.Errorf("couldn't read line: %w", err)
		}

		rawPartHeaders.Write(line)
		rawPartHeaders.Write([]byte("\r\n"))

		if len(line) == 0 {
			break
		}
	}
	mimeHeaders, err := textproto.NewReader(bufio.NewReader(&rawPartHeaders)).ReadMIMEHeader()
	if err != nil {
		return "", fmt.Errorf("couldn't parse MIME headers: %w", err)
	}

	rawDisposition := mimeHeaders["Content-Disposition"]
	if len(rawDisposition) == 0 {
		return "", errors.New("formdata part does not have a Content-Disposition header")
	}

	_, disposition, err := mime.ParseMediaType(rawDisposition[0])
	if err != nil {
		return "", fmt.Errorf("could not parse Content-Disposition header: %w", err)
	}

	partName, ok := disposition["name"]
	if !ok || partName != formDataPartName {
		return "", errors.New("part name is invalid")
	}

	filename, ok := disposition["filename"]
	if !ok || partName != formDataPartName {
		return "", errors.New("no filename in the headers")
	}
	return filename, nil
}

// extractFileSize retrieves the file size (in bytes) from the request headers.
func extractFileSize(headers http.Header) (uint64, error) {
	fileSizeStr := headers.Get(fileSizeHeader)
	if fileSizeStr == "" {
		return 0, fmt.Errorf("%q header is required", fileSizeHeader)
	}

	fileSize, err := strconv.ParseUint(fileSizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid positive integer for %q header", fileSizeStr, fileSizeHeader)
	}

	return fileSize, nil
}

// extractFormDataBoundary extract the boundary for the fromdata from the request headers.
func extractFormDataBoundary(headers http.Header) (string, error) {
	v := headers.Get("Content-Type")
	if v == "" {
		return "", errors.New("no content type header")
	}

	d, params, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		return "", errors.New("not multipart/form-data content type")
	}

	boundary, ok := params["boundary"]
	if !ok {
		return "", errors.New("no boundary specified")
	}

	return boundary, nil
}

// readLine read a line from the file descriptor without buffering.
func readLine(fd int) ([]byte, error) {
	var (
		line = make([]byte, 0, 64)
		c    [1]byte
	)

	for {
		n, err := syscall.Read(fd, c[:])
		if err != nil {
			return nil, fmt.Errorf("could not read byte: %w", err)
		}
		if n != 1 {
			return nil, errors.New("no byte read")
		}

		line = append(line, c[0])

		if len(line) >= 2 && line[len(line)-2] == '\r' && line[len(line)-1] == '\n' {
			return line[:len(line)-2], nil
		}
	}
}
