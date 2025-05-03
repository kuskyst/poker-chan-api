package model

import "github.com/gorilla/websocket"

type Member struct {
	Conn   *websocket.Conn
	Uuid   string
	Name   string
	RoomID string
	Send   chan []byte
}
