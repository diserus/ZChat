package chat

import (
	"testing"

	"github.com/google/uuid"
)

func TestDirectChatPairKeyOrderIndependence(t *testing.T) {
	user1 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	user2 := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	gotA := directChatPairKey(user1, user2)
	gotB := directChatPairKey(user2, user1)

	if gotA != gotB {
		t.Fatalf("pair key must be order independent: %q != %q", gotA, gotB)
	}
}
