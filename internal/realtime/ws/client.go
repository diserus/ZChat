package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"zchat/internal/chat"
	"zchat/internal/realtime"
)

type client struct {
	userID    uuid.UUID
	conn      *websocket.Conn
	send      chan []byte
	subs      map[string]func()
	voiceSubs map[uuid.UUID]struct{}
	mu        *sync.Mutex
	groups    GroupService
	chats     ChatService
	voice     VoiceAudit
	bus       RealtimeBus
	ctx       context.Context
}

func (c *client) readPump() {
	defer func() { _ = c.conn.Close() }()
	c.conn.SetReadLimit(1024 * 16)
	_ = c.conn.SetReadDeadline(timeNow().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.bus.MarkOnline(context.Background(), c.userID)
		return c.conn.SetReadDeadline(timeNow().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = c.bus.MarkOnline(context.Background(), c.userID)
		c.dispatch(data)
	}
}

func (c *client) writePump() {
	ticker := timeTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(timeNow().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.bus.MarkOnline(context.Background(), c.userID)
			_ = c.conn.SetWriteDeadline(timeNow().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *client) close() {
	c.mu.Lock()
	cleanups := make([]func(), 0, len(c.subs))
	for _, fn := range c.subs {
		cleanups = append(cleanups, fn)
	}
	voiceChannels := make([]uuid.UUID, 0, len(c.voiceSubs))
	for id := range c.voiceSubs {
		voiceChannels = append(voiceChannels, id)
	}
	c.subs = map[string]func(){}
	c.voiceSubs = map[uuid.UUID]struct{}{}
	c.mu.Unlock()

	for _, fn := range cleanups {
		fn()
	}
	for _, channelID := range voiceChannels {
		left, err := c.bus.LeaveVoiceChannel(context.Background(), channelID, c.userID)
		if err == nil && left {
			_ = c.bus.PublishVoiceSignaling(context.Background(), realtime.VoiceChannelTopic(channelID), "voice_participant_left", map[string]any{
				"channel_id": channelID.String(),
				"user_id":    c.userID.String(),
			})
		}
	}
}

func (c *client) dispatch(data []byte) {
	var msg clientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("invalid json payload")
		return
	}

	switch msg.Type {
	case "presence_ping":
		_ = c.bus.MarkOnline(context.Background(), c.userID)
		c.sendSystem(serverMessage{Type: "presence_pong"})
	case "subscribe_channel":
		c.handleSubscribeChannel(msg)
	case "unsubscribe_channel":
		c.handleUnsubscribeChannel(msg)
	case "subscribe_direct":
		c.handleSubscribeDirect(msg)
	case "unsubscribe_direct":
		c.handleUnsubscribeDirect(msg)
	case "subscribe_group_presence":
		c.handleSubscribeGroupPresence(msg)
	case "unsubscribe_group_presence":
		c.handleUnsubscribeGroupPresence(msg)
	case "send_channel_message":
		c.handleSendChannelMessage(msg)
	case "send_channel_read":
		c.handleSendChannelRead(msg)
	case "send_direct_message":
		c.handleSendDirectMessage(msg)
	case "send_direct_read":
		c.handleSendDirectRead(msg)
	case "join_voice_channel", "leave_voice_channel",
		"voice_offer", "voice_answer", "voice_ice_candidate",
		"update_voice_state", "moderate_voice_state":
		c.handleVoiceMessage(msg)
	default:
		c.sendError(fmt.Sprintf("unknown event type: %s", msg.Type))
	}
}

func (c *client) handleSubscribeChannel(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	if err = c.groups.AssertChannelMember(context.Background(), channelID, c.userID); err != nil {
		c.sendError("forbidden")
		return
	}
	c.subscribeTopic(realtime.ChannelTopic(channelID))
}

func (c *client) handleUnsubscribeChannel(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	c.unsubscribeTopic(realtime.ChannelTopic(channelID))
}

func (c *client) handleSubscribeDirect(msg clientMessage) {
	directID, err := parseUUID(msg.DirectChatID)
	if err != nil {
		c.sendError("invalid direct_chat_id")
		return
	}
	if err = c.chats.AssertDirectChatMember(context.Background(), directID, c.userID); err != nil {
		c.sendError("forbidden")
		return
	}
	c.subscribeTopic(realtime.DirectTopic(directID))
}

func (c *client) handleUnsubscribeDirect(msg clientMessage) {
	directID, err := parseUUID(msg.DirectChatID)
	if err != nil {
		c.sendError("invalid direct_chat_id")
		return
	}
	c.unsubscribeTopic(realtime.DirectTopic(directID))
}

func (c *client) handleSubscribeGroupPresence(msg clientMessage) {
	groupID, err := parseUUID(msg.GroupID)
	if err != nil {
		c.sendError("invalid group_id")
		return
	}
	groups, err := c.groups.ListUserGroups(context.Background(), c.userID)
	if err != nil {
		c.sendError("failed to verify membership")
		return
	}
	allowed := false
	for _, g := range groups {
		if g.ID == groupID {
			allowed = true
			break
		}
	}
	if !allowed {
		c.sendError("forbidden")
		return
	}
	c.subscribeTopic(realtime.GroupPresenceTopic(groupID))
}

func (c *client) handleUnsubscribeGroupPresence(msg clientMessage) {
	groupID, err := parseUUID(msg.GroupID)
	if err != nil {
		c.sendError("invalid group_id")
		return
	}
	c.unsubscribeTopic(realtime.GroupPresenceTopic(groupID))
}

func (c *client) handleSendChannelMessage(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	out, err := c.chats.SendChannelMessage(context.Background(), chat.SendChannelMessageInput{
		ChannelID: channelID, UserID: c.userID, Content: msg.Content,
	})
	if err != nil {
		c.sendError("failed to send message")
		return
	}
	_ = c.bus.PublishMessageCreated(context.Background(), realtime.ChannelTopic(channelID), mapMessage(out))
}

func (c *client) handleSendChannelRead(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	messageID, err := parseUUID(msg.MessageID)
	if err != nil {
		c.sendError("invalid message_id")
		return
	}
	receipt, err := c.chats.MarkChannelMessageRead(context.Background(), chat.MarkChannelReadInput{
		ChannelID: channelID, UserID: c.userID, MessageID: messageID,
	})
	if err != nil {
		c.sendError("failed to mark message read")
		return
	}
	_ = c.bus.PublishMessageRead(context.Background(), realtime.ChannelTopic(channelID), mapReceipt(receipt))
}

func (c *client) handleSendDirectMessage(msg clientMessage) {
	directID, err := parseUUID(msg.DirectChatID)
	if err != nil {
		c.sendError("invalid direct_chat_id")
		return
	}
	out, err := c.chats.SendDirectMessage(context.Background(), chat.SendDirectMessageInput{
		DirectChatID: directID, UserID: c.userID, Content: msg.Content,
	})
	if err != nil {
		c.sendError("failed to send message")
		return
	}
	_ = c.bus.PublishMessageCreated(context.Background(), realtime.DirectTopic(directID), mapMessage(out))
}

func (c *client) handleSendDirectRead(msg clientMessage) {
	directID, err := parseUUID(msg.DirectChatID)
	if err != nil {
		c.sendError("invalid direct_chat_id")
		return
	}
	messageID, err := parseUUID(msg.MessageID)
	if err != nil {
		c.sendError("invalid message_id")
		return
	}
	receipt, err := c.chats.MarkDirectMessageRead(context.Background(), chat.MarkDirectReadInput{
		DirectChatID: directID, UserID: c.userID, MessageID: messageID,
	})
	if err != nil {
		c.sendError("failed to mark message read")
		return
	}
	_ = c.bus.PublishMessageRead(context.Background(), realtime.DirectTopic(directID), mapReceipt(receipt))
}

func (c *client) subscribeTopic(topic string) {
	c.mu.Lock()
	if _, exists := c.subs[topic]; exists {
		c.mu.Unlock()
		c.sendSystem(serverMessage{Type: "subscribed", Topic: topic})
		return
	}
	if len(c.subs) >= maxWSSubscriptions {
		c.mu.Unlock()
		c.sendError("subscription limit exceeded")
		return
	}
	c.mu.Unlock()

	stream, cleanup, err := c.bus.Subscribe(c.ctx, []string{topic})
	if err != nil {
		c.sendError("failed to subscribe")
		return
	}
	c.mu.Lock()
	c.subs[topic] = cleanup
	c.mu.Unlock()

	go func() {
		for event := range stream {
			select {
			case c.send <- event.Payload:
			case <-c.ctx.Done():
				return
			}
		}
	}()
	c.sendSystem(serverMessage{Type: "subscribed", Topic: topic})
}

func (c *client) unsubscribeTopic(topic string) {
	c.mu.Lock()
	fn, exists := c.subs[topic]
	if exists {
		delete(c.subs, topic)
	}
	c.mu.Unlock()
	if !exists {
		c.sendSystem(serverMessage{Type: "unsubscribed", Topic: topic})
		return
	}
	fn()
	c.sendSystem(serverMessage{Type: "unsubscribed", Topic: topic})
}

func (c *client) sendError(message string) {
	c.sendSystem(serverMessage{Type: "error", Message: message})
}

func (c *client) sendSystem(msg serverMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}
