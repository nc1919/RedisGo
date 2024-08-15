package parser

import (
	"log"
	"net"
)

type Server struct {
	Address string
}

// NewServer initializes a new Server instance with the specified address.
func NewServer(address string) *Server {
	return &Server{
		Address: address,
	}
}

// ListenAndServe starts the TCP server and listens for incoming connections.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Println("Listening on tcp://" + s.Address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}

		log.Println("New connection", conn.RemoteAddr())

		go s.startSession(conn)
	}
}

// startSession handles the client's session. Parses and executes commands and writes
// responses back to the client.
func (s *Server) startSession(conn net.Conn) {
	defer func() {
		log.Println("Closing connection", conn.RemoteAddr())
		conn.Close()
	}()
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from error", err)
		}
	}()

	p := NewParser(conn)
	for {
		cmd, err := p.Command()
		if err != nil {
			log.Println("Error:", err)
			conn.Write([]byte("-ERR " + err.Error() + "\r\n"))
			break
		}
		if !cmd.handle() {
			break
		}
	}
}
