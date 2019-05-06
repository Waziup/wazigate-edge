package main

import (
	"net"
	"os"
)

func notifyDeamon() {
	if name := os.Getenv("NOTIFY_SOCKET"); name != "" {
		socket := &net.UnixAddr{
			Name: name,
			Net:  "unixgram",
		}

		if conn, err := net.DialUnix(socket.Net, nil, socket); err == nil {
			conn.Write([]byte("READY=1"))
			conn.Close()
		}
	}
}
