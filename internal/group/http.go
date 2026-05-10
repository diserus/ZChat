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
