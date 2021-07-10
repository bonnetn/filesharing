package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
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
		return &NotFoundError{Err: fmt.Errorf("resource %q is not found", resourceName)}
	}
	defer fileshare.Conn.Close()

	downloaderConn, _, err := hijackConnection(w)
	if err != nil {
		return err
	}
	defer downloaderConn.Close()

	tcpDownloaderConn, err := extractTCPConn(downloaderConn)
	if err != nil {
		return err
	}

	_, err = tcpDownloaderConn.Write(httpPreludeForFileDownload(fileshare.FileName))
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not send HTTP prelude for file download: %w", err)}
	}

	// NOTE: Previous connection may have buffered some data, before splicing we should send it.
	_, err = tcpDownloaderConn.Write(fileshare.BufferedData)
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not send buffered data: %w", err)}
	}

	// NOTE: On linux, CopyN will use the "splice" syscall which allows very efficient data transfer between conns.
	_, err = io.CopyN(tcpDownloaderConn, fileshare.Conn, fileshare.FileSize)
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not copy data: %v", err)}
	}

	_, err = fileshare.Conn.Write(httpPayloadForSuccessfulUpload)
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not send success response to uploader: %v", err)}
	}

	return nil
}

func httpPreludeForFileDownload(filename string) []byte {
	var buf bytes.Buffer
	buf.WriteString("HTTP/1.1 200 OK\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\r\n", filename))
	buf.WriteString("Content-Type: application/octet-stream\r\n")
	buf.WriteString("\r\n")
	return buf.Bytes()
}

var httpPayloadForSuccessfulUpload = []byte("HTTP/1.1 204 No Content\r\n\r\n")
