package ws

import (
	"context"

	"github.com/google/uuid"

	"zchat/internal/realtime"
	"zchat/internal/voice"
)

func (c *client) handleVoiceMessage(msg clientMessage) {
	switch msg.Type {
	case "join_voice_channel":
		c.handleJoinVoice(msg)
	case "leave_voice_channel":
		c.handleLeaveVoice(msg)
	case "voice_offer", "voice_answer", "voice_ice_candidate":
		c.handleVoiceSignal(msg.Type, msg)
	case "update_voice_state":
		c.handleUpdateVoiceState(msg)
	case "moderate_voice_state":
		c.handleModerateVoiceState(msg)
	}
}

func (c *client) handleJoinVoice(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	if err = c.groups.AssertVoiceChannelMember(context.Background(), channelID, c.userID); err != nil {
		c.sendError("forbidden")
		return
	}
	joined, err := c.bus.JoinVoiceChannel(context.Background(), channelID, c.userID)
	if err != nil {
		c.sendError("failed to join voice channel")
		return
	}
	c.trackVoiceSubscription(channelID)
	topic := realtime.VoiceChannelTopic(channelID)
	c.subscribeTopic(topic)
	snapshot, err := c.buildVoiceParticipantsSnapshot(context.Background(), channelID)
	if err != nil {
		c.sendError("failed to build voice participants snapshot")
		return
	}
	c.sendSystem(serverMessage{
		Type:    "voice_participants_snapshot",
		Payload: map[string]any{"channel_id": channelID.String(), "participants": snapshot},
	})
	if joined {
		_ = c.bus.PublishVoiceSignaling(context.Background(), topic, "voice_participant_joined", map[string]any{
			"channel_id": channelID.String(),
			"user_id":    c.userID.String(),
		})
	}
}

func (c *client) handleLeaveVoice(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	c.untrackVoiceSubscription(channelID)
	topic := realtime.VoiceChannelTopic(channelID)
	left, err := c.bus.LeaveVoiceChannel(context.Background(), channelID, c.userID)
	if err != nil {
		c.sendError("failed to leave voice channel")
		return
	}
	if left {
		_ = c.bus.PublishVoiceSignaling(context.Background(), topic, "voice_participant_left", map[string]any{
			"channel_id": channelID.String(),
			"user_id":    c.userID.String(),
		})
	}
	c.unsubscribeTopic(topic)
}

func (c *client) handleVoiceSignal(eventType string, msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	if err = c.groups.AssertVoiceChannelMember(context.Background(), channelID, c.userID); err != nil {
		c.sendError("forbidden")
		return
	}
	payload := map[string]any{
		"channel_id":   channelID.String(),
		"from_user_id": c.userID.String(),
	}
	if msg.TargetUserID != "" {
		targetUserID, parseErr := parseUUID(msg.TargetUserID)
		if parseErr != nil {
			c.sendError("invalid target_user_id")
			return
		}
		payload["target_user_id"] = targetUserID.String()
	}
	if msg.SDP != "" {
		payload["sdp"] = msg.SDP
	}
	if msg.Candidate != "" {
		payload["candidate"] = msg.Candidate
	}
	if err = c.bus.PublishVoiceSignaling(context.Background(), realtime.VoiceChannelTopic(channelID), eventType, payload); err != nil {
		c.sendError("failed to publish voice signal")
	}
}

func (c *client) handleUpdateVoiceState(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	if err = c.groups.AssertVoiceChannelMember(context.Background(), channelID, c.userID); err != nil {
		c.sendError("forbidden")
		return
	}
	isParticipant, err := c.bus.IsVoiceParticipant(context.Background(), channelID, c.userID)
	if err != nil {
		c.sendError("failed to verify voice participant")
		return
	}
	if !isParticipant {
		c.sendError("user is not in voice channel")
		return
	}
	state, err := c.bus.GetVoiceParticipantState(context.Background(), channelID, c.userID)
	if err != nil {
		c.sendError("failed to get voice state")
		return
	}
	if msg.Muted != nil {
		state.Muted = *msg.Muted
	}
	if msg.Deafened != nil {
		state.Deafened = *msg.Deafened
	}
	if state.MutedByModerator {
		state.Muted = true
	}
	if state.DeafenedByModerator {
		state.Deafened = true
	}
	if msg.HandRaised != nil {
		state.HandRaised = *msg.HandRaised
	}
	if err = c.bus.SetVoiceParticipantState(context.Background(), channelID, c.userID, state); err != nil {
		c.sendError("failed to update voice state")
		return
	}
	_ = c.bus.PublishVoiceSignaling(context.Background(), realtime.VoiceChannelTopic(channelID), "voice_participant_state_updated",
		voiceStatePayload(channelID, c.userID, state))
}

func (c *client) handleModerateVoiceState(msg clientMessage) {
	channelID, err := parseUUID(msg.ChannelID)
	if err != nil {
		c.sendError("invalid channel_id")
		return
	}
	targetUserID, err := parseUUID(msg.TargetUserID)
	if err != nil {
		c.sendError("invalid target_user_id")
		return
	}
	if err = c.groups.AssertVoiceModerationAllowed(context.Background(), channelID, c.userID, targetUserID); err != nil {
		c.sendError("forbidden")
		return
	}
	isParticipant, err := c.bus.IsVoiceParticipant(context.Background(), channelID, targetUserID)
	if err != nil {
		c.sendError("failed to verify voice participant")
		return
	}
	if !isParticipant {
		c.sendError("target user is not in voice channel")
		return
	}
	state, err := c.bus.GetVoiceParticipantState(context.Background(), channelID, targetUserID)
	if err != nil {
		c.sendError("failed to get target voice state")
		return
	}
	if msg.Muted != nil {
		state.Muted = *msg.Muted
		state.MutedByModerator = *msg.Muted
	}
	if msg.Deafened != nil {
		state.Deafened = *msg.Deafened
		state.DeafenedByModerator = *msg.Deafened
	}
	if err = c.bus.SetVoiceParticipantState(context.Background(), channelID, targetUserID, state); err != nil {
		c.sendError("failed to update target voice state")
		return
	}
	if err = c.voice.LogAction(context.Background(), voice.LogActionInput{
		ChannelID:    channelID,
		ActorUserID:  c.userID,
		TargetUserID: targetUserID,
		Muted:        msg.Muted,
		Deafened:     msg.Deafened,
	}); err != nil {
		c.sendError("failed to record moderation action")
		return
	}
	_ = c.bus.PublishVoiceSignaling(context.Background(), realtime.VoiceChannelTopic(channelID), "voice_participant_state_updated",
		voiceStatePayload(channelID, targetUserID, state))
}

func (c *client) buildVoiceParticipantsSnapshot(ctx context.Context, channelID uuid.UUID) ([]map[string]any, error) {
	participants, err := c.bus.ListVoiceParticipants(ctx, channelID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(participants))
	for _, p := range participants {
		state, stateErr := c.bus.GetVoiceParticipantState(ctx, channelID, p)
		if stateErr != nil {
			return nil, stateErr
		}
		out = append(out, voiceStatePayload(channelID, p, state))
	}
	return out, nil
}

func (c *client) trackVoiceSubscription(channelID uuid.UUID) {
	c.mu.Lock()
	c.voiceSubs[channelID] = struct{}{}
	c.mu.Unlock()
}

func (c *client) untrackVoiceSubscription(channelID uuid.UUID) {
	c.mu.Lock()
	delete(c.voiceSubs, channelID)
	c.mu.Unlock()
}

func voiceStatePayload(channelID, userID uuid.UUID, state realtime.VoiceParticipantState) map[string]any {
	return map[string]any{
		"channel_id":            channelID.String(),
		"user_id":               userID.String(),
		"muted":                 state.Muted,
		"deafened":              state.Deafened,
		"hand_raised":           state.HandRaised,
		"muted_by_moderator":    state.MutedByModerator,
		"deafened_by_moderator": state.DeafenedByModerator,
	}
}
