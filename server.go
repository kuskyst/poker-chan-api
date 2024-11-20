package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketアップグレーダー
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// クライアント情報
type Client struct {
	conn   *websocket.Conn
	name   string
	roomID string
	send   chan []byte
}

// 部屋情報
type Room struct {
	ID      string
	clients map[*Client]bool
	mu      sync.Mutex
	votes   map[string]string
}

// ハブ（部屋全体を管理）
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.addClientToRoom(client)
		case client := <-h.unregister:
			h.removeClientFromRoom(client)
		}
	}
}

func (h *Hub) addClientToRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.roomID]
	if !exists {
		room = &Room{
			ID:      client.roomID,
			clients: make(map[*Client]bool),
			votes:   make(map[string]string),
		}
		h.rooms[client.roomID] = room
	}
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	h.broadcastRoomState(room)
}

func (h *Hub) removeClientFromRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.roomID]
	if !exists {
		return
	}
	room.mu.Lock()
	delete(room.clients, client)
	room.mu.Unlock()

	if len(room.clients) == 0 {
		delete(h.rooms, client.roomID)
	} else {
		h.broadcastRoomState(room)
	}
}

func (h *Hub) broadcastRoomState(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	state := map[string]interface{}{
		"clients":   []string{},
		"voteCount": room.votes,
	}

	for client := range room.clients {
		state["clients"] = append(state["clients"].([]string), client.name)
	}

	message, _ := json.Marshal(state)
	for client := range room.clients {
		client.send <- message
	}
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		var data map[string]string
		if err := json.Unmarshal(message, &data); err != nil {
			log.Println("invalid message format")
			continue
		}

		switch data["type"] {
		case "vote":
			h.mu.Lock()
			room := h.rooms[c.roomID]
			room.mu.Lock()
			room.votes[c.name] = data["value"]
			room.mu.Unlock()
			h.mu.Unlock()
			h.broadcastRoomState(room)
		}
	}
}

func (c *Client) writePump() {
	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}
}

func serveWs(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	roomID := r.URL.Query().Get("room")
	name := r.URL.Query().Get("name")
	if roomID == "" || name == "" {
		http.Error(w, "missing room or name", http.StatusBadRequest)
		return
	}

	client := &Client{conn: conn, name: name, roomID: roomID, send: make(chan []byte, 256)}
	h.register <- client

	go client.readPump(h)
	go client.writePump()
}

func main() {
	hub := newHub()
	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
