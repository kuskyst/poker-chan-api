package model

import (
	"encoding/json"
	"log"
	"poker-chan-api/entity"

	"github.com/gorilla/websocket"
)

func (c *Member) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			break
		}

		var message entity.Request
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Invalid message format: %v\n", err)
			continue
		}

		if message.Name != "" {
			c.Name = message.Name
			log.Printf("Client %s registered with name: %s\n", c.Uuid, c.Name)
		}

		if message.Title != "" {
			hub.Rooms[c.RoomID].Title = message.Title
			log.Printf("Client %s registered with title: %s\n", c.Uuid, message.Title)
		}

		if message.Vote != "" {
			hub.Rooms[c.RoomID].Votes[c.Uuid] = message.Vote
			log.Printf("Client %s (%s) voted: %s\n", c.Uuid, c.Name, message.Vote)
		}

		if message.Reset {
			hub.Rooms[c.RoomID].Mu.Lock()
			for memberUuid := range hub.Rooms[c.RoomID].Votes {
				delete(hub.Rooms[c.RoomID].Votes, memberUuid)
			}
			hub.Rooms[c.RoomID].Reveal = false
			hub.Rooms[c.RoomID].Mu.Unlock()
		}

		if message.Reveal {
			hub.Rooms[c.RoomID].Reveal = message.Reveal
		}

		hub.broadcastRoomState(hub.Rooms[c.RoomID])
	}
}

func (c *Member) WritePump() {
	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}
}
