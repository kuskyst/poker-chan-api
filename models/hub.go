package models

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.AddClientToRoom(client)
		case client := <-h.Unregister:
			h.RemoveClientFromRoom(client)
		}
	}
}

func (h *Hub) broadcastRoomState(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	state := map[string]interface{}{
		"type":      "room_state",
		"clients":   []string{},
		"voteCount": len(room.votes),
	}

	for client := range room.clients {
		state["clients"] = append(state["clients"].([]string), client.Name)
	}

	message, _ := json.Marshal(state)
	for client := range room.clients {
		client.Send <- message
	}
}

func (c *Client) ReadPump(h *Hub) {
	defer func() {
		h.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
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
			room := h.Rooms[c.RoomID]
			room.mu.Lock()
			room.votes[c.Name] = data["value"]
			room.mu.Unlock()
			h.mu.Unlock()
			h.broadcastRoomState(room)
		}
	}
}

func (c *Client) WritePump() {
	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
