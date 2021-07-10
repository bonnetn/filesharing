package internal

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// hijackConnection takes ownership of underlying net.Conn.
func hijackConnection(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("responseWrite does not support hijack")
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("error while hijacking the connection: %w", err)
	}
	return conn, buf, nil
}

func extractTCPConn(conn net.Conn) (*net.TCPConn, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("conn is not TCP")
	}
	return tcpConn, nil
}
