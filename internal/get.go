package internal

import (
	"bufio"
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

	downloaderConn, readerWriter, err := hijackConnection(w)
	if err != nil {
		return err
	}
	defer downloaderConn.Close()

	_, err = readerWriter.Write(httpPreludeForFileDownload(fileshare.FileName))
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not send HTTP prelude for file download: %w", err)}
	}

	// Some bytes may are still present in the uploader's Reader buffer, we need to transmit them.
	n, err := transferBufferedBytes(fileshare.Reader, readerWriter.Writer, fileshare.FileSize)
	if err != nil {
		return &LogOnlyError{Err: err}
	}

	bytesLeft := fileshare.FileSize - n

	if bytesLeft > 0 {
		// NOTE: On linux, CopyN will use the "splice" syscall which allows very efficient data transfer between conns.
		// Using the bufio.Reader and bufio.Writer directly prevent the splicing from happening.
		_, err = io.CopyN(downloaderConn, fileshare.Conn, bytesLeft)
		if err != nil {
			return &LogOnlyError{Err: fmt.Errorf("could not copy data: %v", err)}
		}
	}

	_, err = fileshare.Writer.Write(httpPayloadForSuccessfulUpload)
	if err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not send success response to uploader: %v", err)}
	}

	if err := readerWriter.Writer.Flush(); err != nil {
		return &LogOnlyError{Err: fmt.Errorf("could not flush uploader response: %w", err)}
	}

	return nil
}

func transferBufferedBytes(src *bufio.Reader, dst *bufio.Writer, fileSize int64) (int64, error) {
	bytesToTransfer := int64(src.Buffered())
	if bytesToTransfer > fileSize {
		bytesToTransfer = fileSize
	}
	n, err := dst.ReadFrom(io.LimitReader(src, bytesToTransfer))
	if err != nil {
		return n, fmt.Errorf("could not send buffered data: %w", err)
	}

	if err := dst.Flush(); err != nil {
		return n, fmt.Errorf("could not flush bufio: %w", err)
	}
	return n, nil
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
