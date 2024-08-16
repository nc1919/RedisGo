package parser

import (
	"fmt"
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
	if s == nil {
		return fmt.Errorf("Server is nil")
	}
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
		log.Println("Waiting to read data from", conn.RemoteAddr())
		cmd, err := p.Command()
		if err != nil {
			log.Println("Error:", err)
			conn.Write([]byte("-ERR " + err.Error() + "\r\n"))
			break
		}
		log.Println("Command received:", cmd)
		if !cmd.handle() {
			break
		}
	}
}

// func handleConnection(conn net.Conn) {
// 	defer func() {
// 		log.Println("Closing connection from", conn.RemoteAddr())
// 		conn.Close()
// 	}()

// 	buf := make([]byte, 1024)
// 	for {
// 		log.Println("Waiting to read data...")
// 		n, err := conn.Read(buf)
// 		if err != nil {
// 			log.Println("Error reading data:", err)
// 			return
// 		}

// 		log.Printf("Received %d bytes: %s\n", n, string(buf[:n]))
// 		response := "Echo: " + string(buf[:n]) + "\n"
// 		_, err = conn.Write([]byte(response))
// 		if err != nil {
// 			log.Println("Error writing data:", err)
// 			return
// 		}
// 	}
// }
