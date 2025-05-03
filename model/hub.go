package model

import "sync"

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Member
	Unregister chan *Member
	Mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Member),
		Unregister: make(chan *Member),
	}
}
