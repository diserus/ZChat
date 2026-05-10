package group

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}

type CreateChannelRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Type string `json:"type" binding:"required,oneof=text voice"`
}

type GroupResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}

type MemberResponse struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
}

type ChannelResponse struct {
	ID      string `json:"id"`
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
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
