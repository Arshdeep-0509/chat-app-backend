package models

type WSRequest struct {
	Event          string `json:"event"` // "send_message", "typing", "edit_message", "delete_message"
	ConversationID string `json:"conversation_id,omitempty"`
	MessageID      string `json:"message_id,omitempty"`
	Content        string `json:"content,omitempty"`
	IsTyping       bool   `json:"is_typing,omitempty"`
	ParentID       string `json:"parent_id,omitempty"` // For replying to a message
}

type WSResponse struct {
	Event          string      `json:"event"` // "new_message", "typing", "edit_message", "delete_message", "user_status"
	ConversationID string      `json:"conversation_id,omitempty"`
	MessageID      string      `json:"message_id,omitempty"`
	UserID         string      `json:"user_id,omitempty"`
	Content        string      `json:"content,omitempty"`
	IsTyping       bool        `json:"is_typing,omitempty"`
	IsOnline       bool        `json:"is_online"`
	Message        interface{} `json:"message,omitempty"`
}
