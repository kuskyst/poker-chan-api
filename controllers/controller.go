package controllers

import (
	"log"
	"net/http"
	"poker-chan-api/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	hub := models.NewHub()
	go hub.Run()

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

	client := &models.Client{Conn: conn, Name: name, RoomID: roomID, Send: make(chan []byte, 256)}
	hub.Register <- client

	go client.ReadPump(hub)
	go client.WritePump()
}
