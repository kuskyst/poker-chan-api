package model

import "sync"

type Room struct {
	ID      string
	Title   string
	Reveal  bool
	Members map[string]*Member
	Mu      sync.Mutex
	Votes   map[string]string
}
