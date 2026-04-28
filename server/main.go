package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "sync"
    "time"
)

type Server struct {
    listener net.Listener
    port     string
    mu       sync.RWMutex
    rooms    map[string]*Room
}

type Room struct {
    name    string
    clients map[net.Conn]*Client
    mu      sync.RWMutex
}

type Client struct {
    conn net.Conn
    name string
    room *Room
}

type Message struct {
    From    *Client
    Content string
    Time    time.Time
    IsPrivate bool
    TargetName string
}

func NewServer(port string) *Server {
    return &Server{
        port:  port,
        rooms: make(map[string]*Room),
    }
}

func (s *Server) Start() error {
    listener, err := net.Listen("tcp", ":"+s.port)
    if err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    
    s.listener = listener
    log.Printf("Chat server started on port %s", s.port)
    
    s.getOrCreateRoom("general")
    
    return nil
}

func (s *Server) getOrCreateRoom(name string) *Room {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if room, exists := s.rooms[name]; exists {
        return room
    }
    
    room := &Room{
        name:    name,
        clients: make(map[net.Conn]*Client),
    }
    
    s.rooms[name] = room
    log.Printf("Created new room: %s", name)
    return room
}

func (r *Room) broadcast(message string, sender *Client) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    for conn, client := range r.clients {
        if client != sender {
            fmt.Fprintf(conn, "%s\n", message)
        }
    }
}

func (r *Room) addClient(client *Client) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    r.clients[client.conn] = client
    client.room = r
    
    r.broadcast(fmt.Sprintf("%s joined the room", client.name), client)
    

    var members []string
    for _, c := range r.clients {
        if c != client {
            members = append(members, c.name)
        }
    }
    
    if len(members) > 0 {
        fmt.Fprintf(client.conn, "Users in room: %s\n", strings.Join(members, ", "))
    }
}

func (r *Room) removeClient(client *Client) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    delete(r.clients, client.conn)
    r.broadcast(fmt.Sprintf("%s left the room", client.name), client)
}

func (s *Server) AcceptLoop() {
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }
        
        log.Printf("New connection from %s", conn.RemoteAddr())
        go s.handleClient(conn)
    }
}

func (s *Server) handleClient(conn net.Conn) {
    client := &Client{
        conn: conn,
    }
    defer func() {
        if client.room != nil {
            client.room.removeClient(client)
        }
        conn.Close()
        log.Printf("Client %s disconnected", client.name)
    }()
    
    conn.Write([]byte("Welcome! Enter your name: "))
    scanner := bufio.NewScanner(conn)
    
    if !scanner.Scan() {
        return
    }
    client.name = strings.TrimSpace(scanner.Text())
    
    defaultRoom := s.getOrCreateRoom("general")
    defaultRoom.addClient(client)
    
    conn.Write([]byte(fmt.Sprintf("Welcome %s! Commands: /help\n", client.name)))
    
    for scanner.Scan() {
        text := scanner.Text()
        
        if strings.HasPrefix(text, "/") {
            s.handleCommand(client, text)
        } else if text != "" {
            msg := fmt.Sprintf("[%s] %s: %s", 
                time.Now().Format("15:04:05"), 
                client.name, 
                text)
            client.room.broadcast(msg, client)
        }
    }
}

func (s *Server) handleCommand(client *Client, cmd string) {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return
    }
    
    switch parts[0] {
    case "/help":
        fmt.Fprintf(client.conn, `
Commands:
  /users     - Show users in current room
  /join NAME - Switch or create room
  /msg NAME TEXT - Private message
  /quit      - Leave the chat
  /rooms     - List all rooms
`)
        
    case "/users":
        client.room.mu.RLock()
        defer client.room.mu.RUnlock()
        
        var users []string
        for _, c := range client.room.clients {
            if c != client {
                users = append(users, c.name)
            }
        }
        fmt.Fprintf(client.conn, "Users in %s: %s\n", 
            client.room.name, 
            strings.Join(users, ", "))
        
    case "/rooms":
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        var rooms []string
        for name, room := range s.rooms {
            room.mu.RLock()
            count := len(room.clients)
            room.mu.RUnlock()
            rooms = append(rooms, fmt.Sprintf("%s(%d)", name, count))
        }
        fmt.Fprintf(client.conn, "Rooms: %s\n", strings.Join(rooms, ", "))
        
    case "/join":
        if len(parts) < 2 {
            fmt.Fprintf(client.conn, "Usage: /join ROOM_NAME\n")
            return
        }
        
        newRoomName := parts[1]
        
        if client.room != nil {
            client.room.removeClient(client)
        }
        
        newRoom := s.getOrCreateRoom(newRoomName)
        newRoom.addClient(client)
        fmt.Fprintf(client.conn, "Switched to room: %s\n", newRoomName)
        
    case "/msg":
        if len(parts) < 3 {
            fmt.Fprintf(client.conn, "Usage: /msg USERNAME MESSAGE\n")
            return
        }
        
        targetName := parts[1]
        message := strings.Join(parts[2:], " ")
        
        client.room.mu.RLock()
        defer client.room.mu.RUnlock()
        
        for _, c := range client.room.clients {
            if c.name == targetName && c != client {
                fmt.Fprintf(c.conn, "\n[PM from %s]: %s\n", client.name, message)
                fmt.Fprintf(client.conn, "[PM to %s]: %s\n", targetName, message)
                return
            }
        }
        fmt.Fprintf(client.conn, "User %s not found in current room\n", targetName)
        
    case "/quit":
        fmt.Fprintf(client.conn, "Goodbye!\n")
        client.conn.Close()
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
    
    log.Println("Server ready. Connect with: telnet localhost 8080")
    server.AcceptLoop()
}