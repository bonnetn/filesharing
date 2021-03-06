package internal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
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

	conn, readWriter, err := hijackConnection(w)
	if err != nil {
		return err
	}

	if err := skipBoundary(readWriter.Reader, boundary); err != nil {
		return &LogOnlyError{Err: &BadRequestError{Err: err}}
	}

	filename, err := skipFormDataHeaders(readWriter.Reader)
	if err != nil {
		return &LogOnlyError{Err: err}
	}

	c := PendingFileshare{
		Conn:     conn,
		Reader:   readWriter.Reader,
		Writer:   readWriter.Writer,
		FileSize: fileSize,
		FileName: filename,
	}

	if !o.repository.Set(resourceName, c) {
		return &LogOnlyError{Err: fmt.Errorf("there is already a resource with name %q", resourceName)}
	}
	return nil
}

// skipBoundary skips the first line of formdata.
func skipBoundary(reader *bufio.Reader, boundary string) error {
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("couldn't read line: %w", err)
	}

	if strings.Compare(firstLine, "--"+boundary+"\r\n") != 0 {
		return fmt.Errorf("wrong boundary, expected %q, got %q", boundary, string(firstLine))
	}

	return nil
}

// skipFormDataHeaders skips the formdata headers.
func skipFormDataHeaders(reader *bufio.Reader) (string, error) {
	var rawPartHeaders bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return "", fmt.Errorf("couldn't read line: %w", err)
		}

		rawPartHeaders.Write(line)

		if bytes.Compare(line, []byte("\r\n")) == 0 {
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
func extractFileSize(headers http.Header) (int64, error) {
	fileSizeStr := headers.Get(fileSizeHeader)
	if fileSizeStr == "" {
		return 0, fmt.Errorf("%q header is required", fileSizeHeader)
	}

	fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid positive integer for %q header: %w", fileSizeStr, fileSizeHeader, err)
	}

	if fileSize < 0 {
		return 0, errors.New("file size must be a positive integer")
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
