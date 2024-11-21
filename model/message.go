package model

type Message struct {
	Name  string `json:"name,omitempty"`
	Vote  string `json:"vote,omitempty"`
	Reset bool   `json:"reset,omitempty"`
}
