package model

type Message struct {
	Title string `json:"title,omitempty"`
	Name  string `json:"name,omitempty"`
	Vote  string `json:"vote,omitempty"`
	Reset bool   `json:"reset,omitempty"`
}
