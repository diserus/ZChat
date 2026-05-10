package group

import "testing"

func TestRoleValid(t *testing.T) {
	cases := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, true},
		{Role("guest"), false},
		{Role(""), false},
	}
	for _, c := range cases {
		if got := c.role.Valid(); got != c.want {
			t.Errorf("%q.Valid() = %v, want %v", c.role, got, c.want)
		}
	}
}

func TestChannelValidate(t *testing.T) {
	cases := []struct {
		name    string
		channel Channel
		wantErr bool
	}{
		{"text ok", Channel{Name: "general", Type: ChannelTypeText}, false},
		{"voice ok", Channel{Name: "lobby", Type: ChannelTypeVoice}, false},
		{"empty name", Channel{Name: "", Type: ChannelTypeText}, true},
		{"bad type", Channel{Name: "x", Type: ChannelType("video")}, true},
	}
	for _, c := range cases {
		err := c.channel.Validate()
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr = %v", c.name, err, c.wantErr)
		}
	}
}
