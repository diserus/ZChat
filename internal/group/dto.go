package group

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150" example:"My Awesome Group" description:"Group name, 2-150 chars"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member" example:"admin" enums:"admin,member" description:"New role for the user"`
}

type CreateChannelRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100" example:"general" description:"Channel name"`
	Type string `json:"type" binding:"required,oneof=text voice" example:"text" enums:"text,voice" description:"Channel type"`
}

type GroupResponse struct {
	ID      string `json:"id" example:"group-123"`
	Name    string `json:"name" example:"My Awesome Group"`
	OwnerID string `json:"owner_id" example:"user-owner-567"`
}

type MemberResponse struct {
	GroupID string `json:"group_id" example:"group-123"`
	UserID  string `json:"user_id" example:"user-789"`
	Role    string `json:"role" example:"admin" enums:"admin,member"`
}

type ChannelResponse struct {
	ID      string `json:"id" example:"channel-456"`
	GroupID string `json:"group_id" example:"group-123"`
	Name    string `json:"name" example:"general"`
	Type    string `json:"type" example:"text" enums:"text,voice"`
}

func toGroupResponse(g *Group) GroupResponse {
	return GroupResponse{ID: g.ID.String(), Name: g.Name, OwnerID: g.OwnerID.String()}
}

func toGroupsResponse(gs []Group) []GroupResponse {
	out := make([]GroupResponse, 0, len(gs))
	for i := range gs {
		out = append(out, toGroupResponse(&gs[i]))
	}
	return out
}

func toMemberResponse(m *Member) MemberResponse {
	return MemberResponse{GroupID: m.GroupID.String(), UserID: m.UserID.String(), Role: string(m.Role)}
}

func toMembersResponse(ms []Member) []MemberResponse {
	out := make([]MemberResponse, 0, len(ms))
	for i := range ms {
		out = append(out, toMemberResponse(&ms[i]))
	}
	return out
}

func toChannelResponse(c *Channel) ChannelResponse {
	return ChannelResponse{ID: c.ID.String(), GroupID: c.GroupID.String(), Name: c.Name, Type: string(c.Type)}
}

func toChannelsResponse(cs []Channel) []ChannelResponse {
	out := make([]ChannelResponse, 0, len(cs))
	for i := range cs {
		out = append(out, toChannelResponse(&cs[i]))
	}
	return out
}
