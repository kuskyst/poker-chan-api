package models

import "github.com/gorilla/websocket"

type Client struct {
	Conn   *websocket.Conn
	Name   string
	RoomID string
	Send   chan []byte
}

func (h *Hub) AddClientToRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.Rooms[client.RoomID]
	if !exists {
		room = &Room{
			ID:      client.RoomID,
			clients: make(map[*Client]bool),
			votes:   make(map[string]string),
		}
		h.Rooms[client.RoomID] = room
	}
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	h.broadcastRoomState(room)
}

func (h *Hub) RemoveClientFromRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.Rooms[client.RoomID]
	if !exists {
		return
	}
	room.mu.Lock()
	delete(room.clients, client)
	room.mu.Unlock()

	if len(room.clients) == 0 {
		delete(h.Rooms, client.RoomID)
	} else {
		h.broadcastRoomState(room)
	}
}
