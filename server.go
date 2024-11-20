package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Member struct {
	conn   *websocket.Conn
	name   string
	roomID string
	send   chan []byte
}

type Room struct {
	ID      string
	members map[*Member]bool
	mu      sync.Mutex
	votes   map[string]string
}

type Hub struct {
	rooms      map[string]*Room
	register   chan *Member
	unregister chan *Member
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Member),
		unregister: make(chan *Member),
	}
}

func (h *Hub) run() {
	for {
		select {
		case member := <-h.register:
			h.addmemberToRoom(member)
		case member := <-h.unregister:
			h.removememberFromRoom(member)
		}
	}
}

func (h *Hub) addmemberToRoom(member *Member) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[member.roomID]
	if !exists {
		room = &Room{
			ID:      member.roomID,
			members: make(map[*Member]bool),
			votes:   make(map[string]string),
		}
		h.rooms[member.roomID] = room
	}
	room.mu.Lock()
	room.members[member] = true
	room.mu.Unlock()

	h.broadcastRoomState(room)
}

func (h *Hub) removememberFromRoom(member *Member) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[member.roomID]
	if !exists {
		return
	}
	room.mu.Lock()
	delete(room.members, member)
	room.mu.Unlock()

	if len(room.members) == 0 {
		delete(h.rooms, member.roomID)
	} else {
		h.broadcastRoomState(room)
	}
}

func (h *Hub) broadcastRoomState(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	state := map[string]interface{}{
		"members": []string{},
		"votes":   room.votes,
	}

	for member := range room.members {
		state["members"] = append(state["members"].([]string), member.name)
	}

	message, _ := json.Marshal(state)
	for member := range room.members {
		member.send <- message
	}
}

func (c *Member) readPump(h *Hub) {
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

func (c *Member) writePump() {
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

	member := &Member{conn: conn, name: name, roomID: roomID, send: make(chan []byte, 256)}
	h.register <- member

	go member.readPump(h)
	go member.writePump()
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
