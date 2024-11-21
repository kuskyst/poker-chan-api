package main

import (
	"encoding/json"
	"log"
	"net/http"
	"poker-chan-api/model"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Member struct {
	conn   *websocket.Conn
	uuid   string
	name   string
	roomID string
	send   chan []byte
}

type Room struct {
	ID      string
	title   string
	reveal  bool
	members map[string]*Member
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
			members: make(map[string]*Member),
			votes:   make(map[string]string),
		}
		h.rooms[member.roomID] = room
	}

	room.mu.Lock()
	room.members[member.uuid] = member
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
	delete(room.members, member.uuid)
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

	members := make([]map[string]string, 0)
	for _, member := range room.members {
		members = append(members, map[string]string{
			"uuid": member.uuid,
			"name": member.name,
		})
	}

	votes := room.votes

	state := map[string]interface{}{
		"reveal":  room.reveal,
		"title":   room.title,
		"members": members,
		"votes":   votes,
	}

	message, _ := json.Marshal(state)

	for _, member := range room.members {
		member.send <- message
	}
}

func (c *Member) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			break
		}

		var message model.Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Invalid message format: %v\n", err)
			continue
		}

		if message.Name != "" {
			c.name = message.Name
			log.Printf("Client %s registered with name: %s\n", c.uuid, c.name)
		}

		if message.Title != "" {
			hub.rooms[c.roomID].title = message.Title
			log.Printf("Client %s registered with title: %s\n", c.uuid, message.Title)
		}

		if message.Vote != "" {
			hub.rooms[c.roomID].votes[c.uuid] = message.Vote
			log.Printf("Client %s (%s) voted: %s\n", c.uuid, c.name, message.Vote)
		}

		if message.Reset {
			hub.rooms[c.roomID].mu.Lock()
			for memberUuid := range hub.rooms[c.roomID].votes {
				delete(hub.rooms[c.roomID].votes, memberUuid)
			}
			hub.rooms[c.roomID].reveal = false
			hub.rooms[c.roomID].mu.Unlock()
		}

		if message.Reveal {
			hub.rooms[c.roomID].reveal = message.Reveal
		}

		hub.broadcastRoomState(hub.rooms[c.roomID])
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

	roomID := r.URL.Query().Get("id")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	clientUUID := uuid.New().String()
	member := &Member{
		uuid:   clientUUID,
		conn:   conn,
		roomID: roomID,
		send:   make(chan []byte, 256),
	}

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
