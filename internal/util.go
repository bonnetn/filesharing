package internal

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

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

func extractTCPConn(conn net.Conn) (*net.TCPConn, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("conn is not TCP")
	}
	return tcpConn, nil
}
