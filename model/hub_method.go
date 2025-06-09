package model

import (
	"encoding/json"
)

func (h *Hub) Run() {
	for {
		select {
		case member := <-h.Register:
			h.addMemberToRoom(member)
		case member := <-h.Unregister:
			h.removeMemberFromRoom(member)
		}
	}
}

func (h *Hub) addMemberToRoom(member *Member) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	room, exists := h.Rooms[member.RoomID]
	if !exists {
		room = &Room{
			ID:      member.RoomID,
			Members: make(map[string]*Member),
			Votes:   make(map[string]string),
		}
		h.Rooms[member.RoomID] = room
	}

	room.Mu.Lock()
	room.Members[member.Uuid] = member
	room.Mu.Unlock()

	h.broadcastRoomState(room)
}

func (h *Hub) removeMemberFromRoom(member *Member) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	room, exists := h.Rooms[member.RoomID]
	if !exists {
		return
	}
	room.Mu.Lock()
	delete(room.Members, member.Uuid)
	delete(room.Votes, member.Uuid)
	room.Mu.Unlock()

	if len(room.Members) == 0 {
		delete(h.Rooms, member.RoomID)
	} else {
		h.broadcastRoomState(room)
	}
}

func (h *Hub) broadcastRoomState(room *Room) {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	members := make([]map[string]string, 0)
	for _, member := range room.Members {
		members = append(members, map[string]string{
			"uuid": member.Uuid,
			"name": member.Name,
		})
	}

	votes := room.Votes

	state := map[string]interface{}{
		"reveal":  room.Reveal,
		"title":   room.Title,
		"members": members,
		"votes":   votes,
	}

	message, _ := json.Marshal(state)

	for _, member := range room.Members {
		member.Send <- message
	}
}
