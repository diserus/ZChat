package realtime

import "github.com/google/uuid"

// Topic builders define the wire-level pub/sub naming contract that the
// realtime layer exposes to all bounded contexts.

func ChannelTopic(channelID uuid.UUID) string {
	return "rt:channel:" + channelID.String()
}

func DirectTopic(directChatID uuid.UUID) string {
	return "rt:direct:" + directChatID.String()
}

func GroupPresenceTopic(groupID uuid.UUID) string {
	return "rt:presence:group:" + groupID.String()
}

func VoiceChannelTopic(channelID uuid.UUID) string {
	return "rt:voice:channel:" + channelID.String()
}
