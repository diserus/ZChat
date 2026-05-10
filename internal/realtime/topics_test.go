package realtime

import (
	"testing"

	"github.com/google/uuid"
)

func TestTopicBuilders(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"channel", ChannelTopic(id), "rt:channel:" + id.String()},
		{"direct", DirectTopic(id), "rt:direct:" + id.String()},
		{"group_presence", GroupPresenceTopic(id), "rt:presence:group:" + id.String()},
		{"voice_channel", VoiceChannelTopic(id), "rt:voice:channel:" + id.String()},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}
