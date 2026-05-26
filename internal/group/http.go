package group

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zchat/internal/auth"
	"zchat/internal/httpapi"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(_ *gin.RouterGroup, protected *gin.RouterGroup) {
	protected.GET("/groups", h.listGroups)
	protected.POST("/groups", h.createGroup)
	protected.GET("/groups/:group_id/members", h.listMembers)
	protected.POST("/groups/:group_id/members", h.addMember)
	protected.PATCH("/groups/:group_id/members/:user_id/role", h.updateMemberRole)
	protected.DELETE("/groups/:group_id/members/:user_id", h.removeMember)
	protected.GET("/groups/:group_id/channels", h.listChannels)
	protected.POST("/groups/:group_id/channels", h.createChannel)
}

// CreateGroup godoc
// @Summary      Create a new group
// @Tags         groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body CreateGroupRequest true "Group name"
// @Success      201  {object}  GroupResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /groups [post]
func (h *Handler) createGroup(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	g, err := h.svc.CreateGroup(c.Request.Context(), CreateGroupInput{OwnerID: userID, Name: req.Name})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toGroupResponse(g))
}

// ListGroups godoc
// @Summary      List user groups
// @Description  Returns all groups the authenticated user belongs to
// @Tags         groups
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "{\"groups\": [...]}"
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /groups [get]
func (h *Handler) listGroups(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groups, err := h.svc.ListUserGroups(c.Request.Context(), userID)
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": toGroupsResponse(groups)})
}

// ListMembers godoc
// @Summary      List group members
// @Tags         groups
// @Security     BearerAuth
// @Param        group_id path string true "Group ID"
// @Success      200  {object}  map[string]interface{}  "{\"members\": [...]}"
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Failure      403  {object}  httpapi.ErrorResponse  "Not a member"
// @Router       /groups/{group_id}/members [get]
func (h *Handler) listMembers(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	members, err := h.svc.ListMembers(c.Request.Context(), groupID, userID)
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": toMembersResponse(members)})
}

// AddMember godoc
// @Summary      Add member to group
// @Tags         groups
// @Security     BearerAuth
// @Accept       json
// @Param        group_id path string true "Group ID"
// @Param        request body AddMemberRequest true "User ID to add"
// @Success      200  {object}  map[string]interface{}  "{\"message\":\"member added\"}"
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Failure      403  {object}  httpapi.ErrorResponse  "Insufficient role"
// @Router       /groups/{group_id}/members [post]
func (h *Handler) addMember(c *gin.Context) {
	actorID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req AddMemberRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	targetID, err := httpapi.ParseUUIDParam(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	if err = h.svc.AddMember(c.Request.Context(), AddMemberInput{
		GroupID: groupID, ActorUserID: actorID, TargetUserID: targetID,
	}); err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member added"})
}

// UpdateMemberRole godoc
// @Summary      Update member role (admin or member)
// @Tags         groups
// @Security     BearerAuth
// @Accept       json
// @Param        group_id path string true "Group ID"
// @Param        user_id path string true "User ID"
// @Param        request body UpdateMemberRoleRequest true "New role"
// @Success      200  {object}  map[string]interface{}  "{\"message\":\"role updated\"}"
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Failure      403  {object}  httpapi.ErrorResponse
// @Router       /groups/{group_id}/members/{user_id}/role [patch]
func (h *Handler) updateMemberRole(c *gin.Context) {
	actorID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	targetID, err := httpapi.ParseUUIDParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req UpdateMemberRoleRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	if err = h.svc.UpdateMemberRole(c.Request.Context(), UpdateMemberRoleInput{
		GroupID: groupID, ActorUserID: actorID, TargetUserID: targetID, Role: Role(req.Role),
	}); err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// RemoveMember godoc
// @Summary      Remove member from group
// @Tags         groups
// @Security     BearerAuth
// @Param        group_id path string true "Group ID"
// @Param        user_id path string true "User ID"
// @Success      200  {object}  map[string]interface{}  "{\"message\":\"member removed\"}"
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Failure      403  {object}  httpapi.ErrorResponse
// @Router       /groups/{group_id}/members/{user_id} [delete]
func (h *Handler) removeMember(c *gin.Context) {
	actorID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	targetID, err := httpapi.ParseUUIDParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	if err = h.svc.RemoveMember(c.Request.Context(), RemoveMemberInput{
		GroupID: groupID, ActorUserID: actorID, TargetUserID: targetID,
	}); err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// CreateChannel godoc
// @Summary      Create a channel (text or voice)
// @Tags         channels
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        group_id path string true "Group ID"
// @Param        request body CreateChannelRequest true "Channel name and type"
// @Success      201  {object}  ChannelResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /groups/{group_id}/channels [post]
func (h *Handler) createChannel(c *gin.Context) {
	actorID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req CreateChannelRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	ch, err := h.svc.CreateChannel(c.Request.Context(), CreateChannelInput{
		GroupID: groupID, ActorUserID: actorID, Name: req.Name, Type: ChannelType(req.Type),
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toChannelResponse(ch))
}

// ListChannels godoc
// @Summary      List channels in group
// @Tags         channels
// @Security     BearerAuth
// @Param        group_id path string true "Group ID"
// @Success      200  {object}  map[string]interface{}  "{\"channels\": [...]}"
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /groups/{group_id}/channels [get]
func (h *Handler) listChannels(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	channels, err := h.svc.ListChannels(c.Request.Context(), groupID, userID)
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": toChannelsResponse(channels)})
}
