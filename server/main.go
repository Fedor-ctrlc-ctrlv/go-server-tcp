package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
)

type Server struct {
    listener net.Listener
    port     string
}

type Client struct {
    conn net.Conn
    name string
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
        
        client := &Client{
            conn: conn,
        }
        
        log.Printf("New connection from %s", conn.RemoteAddr())
        go s.handleClient(client)
    }
}

func (s *Server) handleClient(client *Client) {
    defer client.conn.Close()
    defer log.Printf("Client %s disconnected", client.conn.RemoteAddr())
    

    client.conn.Write([]byte("Enter your name: "))
    scanner := bufio.NewScanner(client.conn)
    
    if scanner.Scan() {
        client.name = strings.TrimSpace(scanner.Text())
        log.Printf("Client %s identified as %s", client.conn.RemoteAddr(), client.name)
        client.conn.Write([]byte(fmt.Sprintf("Hello %s! Commands: /echo, /quit\n", client.name)))
    }
    
    for scanner.Scan() {
        text := scanner.Text()
        
        switch text {
        case "/quit":
            client.conn.Write([]byte("Goodbye!\n"))
            return
            
        case "/echo":
            client.conn.Write([]byte("Send message to echo: "))
            if scanner.Scan() {
                client.conn.Write([]byte(fmt.Sprintf("Echo: %s\n", scanner.Text())))
            }
            
        default:
            client.conn.Write([]byte(fmt.Sprintf("You said: %s\n", text)))
        }
    }
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