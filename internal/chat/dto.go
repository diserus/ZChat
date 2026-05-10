package chat

import "time"

type CreateDirectChatRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=4000"`
}

type MarkMessageReadRequest struct {
	MessageID string `json:"message_id" binding:"required,uuid"`
}

type DirectChatResponse struct {
	ID string `json:"id"`
}

type MessageResponse struct {
	ID           string     `json:"id"`
	SenderID     string     `json:"sender_id"`
	ChannelID    *string    `json:"channel_id,omitempty"`
	DirectChatID *string    `json:"direct_chat_id,omitempty"`
	Content      string     `json:"content"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

type ReceiptResponse struct {
	MessageID   string     `json:"message_id"`
	UserID      string     `json:"user_id"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
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
