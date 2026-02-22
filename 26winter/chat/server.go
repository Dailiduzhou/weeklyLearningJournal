package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message 定义消息结构
type Message struct {
	Type      string    `json:"type"`
	Username  string    `json:"username,omitempty"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Usernames []string  `json:"usernames,omitempty"`
}

type Client struct {
	conn     *websocket.Conn
	send     chan Message
	username string
	hub      *Hub
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	usernames  map[string]bool
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		usernames:  make(map[string]bool),
	}
}

// Helper: 统一处理消息广播，避免重复代码
func (h *Hub) broadcastToClients(msg Message) {
	log.Printf("Broadcasting message type: %s to %d clients", msg.Type, len(h.clients))
	for client := range h.clients {
		select {
		case client.send <- msg:
			// 发送成功
		default:
			// 发送阻塞，假设客户端断开或太慢，记录日志（实际生产中可能需要断开连接）
			log.Printf("  - Failed to send to %s (buffer full)", client.username)
		}
	}
}

func (h *Hub) run() {
	log.Println("Hub started")
	for {
		select {
		case client := <-h.register:
			log.Printf("Registering client: %s", client.username)
			h.clients[client] = true
			h.usernames[client.username] = true
			log.Printf("  Client count after register: %d", len(h.clients))

			// 构造登录消息
			loginMsg := Message{
				Type:      "login",
				Username:  client.username,
				Timestamp: time.Now(),
			}
			// 构造用户列表消息
			userlistMsg := Message{
				Type:      "userlist",
				Usernames: getUsernames(h), // 安全：在 Hub goroutine 内访问 map
				Timestamp: time.Now(),
			}

			h.broadcastToClients(loginMsg)
			h.broadcastToClients(userlistMsg)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.usernames, client.username)
				close(client.send)

				logoutMsg := Message{
					Type:      "logout",
					Username:  client.username,
					Timestamp: time.Now(),
				}
				userlistMsg := Message{
					Type:      "userlist",
					Usernames: getUsernames(h), // 安全
					Timestamp: time.Now(),
				}

				h.broadcastToClients(logoutMsg)
				h.broadcastToClients(userlistMsg)
			}

		case message := <-h.broadcast:
			h.broadcastToClients(message)
		}
	}
}

func getUsernames(h *Hub) []string {
	var usernames []string
	for username := range h.usernames {
		usernames = append(usernames, username)
	}
	return usernames
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096) // 建议：设置读取上限
	// 建议：设置读取超时 (SetReadDeadline)，配合 Ping/Pong 使用
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error from %s: %v", c.username, err)
			}
			break
		}
		msg.Username = c.username
		msg.Timestamp = time.Now()
		log.Printf("Received message from %s: %s", c.username, msg.Content)
		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second) // 略小于 ReadDeadline
	defer func() {
		ticker.Stop()
		// 可以在这里处理关闭错误，通常忽略
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// [ADDED] 增加 SetWriteDeadline 错误检查
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("SetWriteDeadline error for %s: %v", c.username, err)
				return
			}

			if !ok {
				// Hub 关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("Write error to %s: %v", c.username, err)
				return
			}
			// log.Printf("Sent message to %s: %s", c.username, message.Type)

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("SetWriteDeadline (Ping) error for %s: %v", c.username, err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error to %s: %v", c.username, err)
				return
			}
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Anonymous"
	}

	log.Printf("New connection from %s", username)

	client := &Client{
		conn:     conn,
		send:     make(chan Message, 256), // 适当减小 buffer，防止内存占用过大
		username: username,
		hub:      hub,
	}

	// 先启动 Pump，再注册
	go client.writePump()
	go client.readPump()

	// 注册客户端
	hub.register <- client
}

func main() {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	log.Println("Server started on :8080")
	// ListenAndServe 返回的 error 应该被处理（log.Fatal 已经处理了）
	log.Fatal(http.ListenAndServe(":8080", nil))
}
