package models

import "sync"

type Room struct {
	ID      string
	clients map[*Client]bool
	mu      sync.Mutex
	votes   map[string]string
}
