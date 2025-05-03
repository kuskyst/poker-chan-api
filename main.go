package main

import (
	"log"
	"net/http"

	"poker-chan-api/model"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func serveWs(h *model.Hub, w http.ResponseWriter, r *http.Request) {
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
	member := &model.Member{
		Uuid:   clientUUID,
		Conn:   conn,
		RoomID: roomID,
		Send:   make(chan []byte, 256),
	}

	h.Register <- member

	go member.ReadPump(h)
	go member.WritePump()
}

func main() {
	hub := model.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
