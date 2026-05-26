package chat

import "time"

// CreateDirectChatRequest for POST /direct-chats
type CreateDirectChatRequest struct {
	// User ID of the other participant (UUID format)
	UserID string `json:"user_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000" description:"Target user ID"`
}

// SendMessageRequest for POST /channels/:channel_id/messages and direct chats
type SendMessageRequest struct {
	// Message content, min 1, max 4000 characters
	Content string `json:"content" binding:"required,min=1,max=4000" example:"Hello, world!" description:"Text content of the message"`
}

// MarkMessageReadRequest for POST /channels/:channel_id/read and direct reads
type MarkMessageReadRequest struct {
	// ID of the message being marked as read
	MessageID string `json:"message_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// DirectChatResponse returned for direct chat creation/GET
type DirectChatResponse struct {
	ID string `json:"id" example:"direct-12345" description:"Direct chat unique identifier"`
}

// MessageResponse for messages (channel or direct)
type MessageResponse struct {
	ID           string     `json:"id" example:"msg-123" description:"Message UUID"`
	SenderID     string     `json:"sender_id" example:"user-789" description:"User ID of the sender"`
	ChannelID    *string    `json:"channel_id,omitempty" description:"If the message belongs to a group channel, the channel ID"`
	DirectChatID *string    `json:"direct_chat_id,omitempty" description:"If the message belongs to a direct chat, the direct chat ID"`
	Content      string     `json:"content" example:"Hello!" description:"Message text"`
	CreatedAt    *time.Time `json:"created_at,omitempty" example:"2025-01-15T12:34:56Z"`
}

// ReceiptResponse for message delivery/read receipts
type ReceiptResponse struct {
	MessageID   string     `json:"message_id" example:"msg-123"`
	UserID      string     `json:"user_id" example:"user-789"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" example:"2025-01-15T12:34:56Z"`
	ReadAt      *time.Time `json:"read_at,omitempty" example:"2025-01-15T12:35:00Z"`
}

func toDirectChatResponse(c *DirectChat) DirectChatResponse {
	return DirectChatResponse{ID: c.ID.String()}
}

func toMessageResponse(m *Message) MessageResponse {
	var channelID *string
	if m.ChannelID != nil {
		v := m.ChannelID.String()
		channelID = &v
	}
	var directChatID *string
	if m.DirectChatID != nil {
		v := m.DirectChatID.String()
		directChatID = &v
	}
	var createdAt *time.Time
	if !m.CreatedAt.IsZero() {
		t := m.CreatedAt
		createdAt = &t
	}
	return MessageResponse{
		ID:           m.ID.String(),
		SenderID:     m.SenderID.String(),
		ChannelID:    channelID,
		DirectChatID: directChatID,
		Content:      m.Content,
		CreatedAt:    createdAt,
	}
}

func toMessagesResponse(ms []Message) []MessageResponse {
	out := make([]MessageResponse, 0, len(ms))
	for i := range ms {
		out = append(out, toMessageResponse(&ms[i]))
	}
	return out
}

func toReceiptResponse(r *Receipt) ReceiptResponse {
	return ReceiptResponse{
		MessageID:   r.MessageID.String(),
		UserID:      r.UserID.String(),
		DeliveredAt: r.DeliveredAt,
		ReadAt:      r.ReadAt,
	}
}
