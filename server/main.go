package main

import (
    "fmt"
    "log"
    "net"
)

type Server struct {
    listener net.Listener
    port     string
}

func NewServer(port string) *Server {
    return &Server{
        port: port,
    }
}

func (s *Server) Start() error {
    listener, err := net.Listen("tcp", ":"+s.port)
    if err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    
    s.listener = listener
    log.Printf("Server listening on port %s", s.port)
    
    return nil
}

func (s *Server) AcceptLoop() {
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }
        
        log.Printf("New connection from %s", conn.RemoteAddr())
        go s.handleConnection(conn)
    }
}

func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()
    defer log.Printf("Connection closed from %s", conn.RemoteAddr())

    conn.Write([]byte("Welcome to TCP server!\n"))
}

func (s *Server) Stop() error {
    return s.listener.Close()
}

func main() {
    server := NewServer("8080")
    
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
    
    server.AcceptLoop()
}