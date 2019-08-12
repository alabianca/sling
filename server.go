package main

import (
	"bytes"
	"io"
	"net"
)

type Server struct {
	Addr     string
	Data     chan []byte
	listener net.Listener
}

func (s *Server) ListenForSingleConnection() chan net.Conn {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}
	out := make(chan net.Conn)
	s.listener = l

	go func(res chan net.Conn) {
		conn, err := s.listener.Accept()
		defer s.listener.Close()
		if err != nil {
			panic(err)
		}

		res <- conn
	}(out)

	return out

}

func (s *Server) handleConnection(c net.Conn) {
	buf := new(bytes.Buffer)
	io.Copy(buf, c)

	s.Data <- buf.Bytes()
}
