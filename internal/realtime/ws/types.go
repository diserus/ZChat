package ws

import "time"

const (
	writeWait          = 10 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = 30 * time.Second
	maxWSSubscriptions = 128
)

type clientMessage struct {
	Type         string `json:"type"`
	ChannelID    string `json:"channel_id,omitempty"`
	DirectChatID string `json:"direct_chat_id,omitempty"`
	GroupID      string `json:"group_id,omitempty"`
	MessageID    string `json:"message_id,omitempty"`
	TargetUserID string `json:"target_user_id,omitempty"`
	SDP          string `json:"sdp,omitempty"`
	Candidate    string `json:"candidate,omitempty"`
	Muted        *bool  `json:"muted,omitempty"`
	Deafened     *bool  `json:"deafened,omitempty"`
	HandRaised   *bool  `json:"hand_raised,omitempty"`
	Content      string `json:"content,omitempty"`
}

type serverMessage struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Payload any    `json:"payload,omitempty"`
}
