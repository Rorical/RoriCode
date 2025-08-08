package models

type MessageType int

const (
	System MessageType = iota
	User
	Assistant
	Program
)

type Message struct {
	Content string
	Type    MessageType
}
