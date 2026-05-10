package chat

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"zchat/internal/auth"
	"zchat/internal/httpapi"
	"zchat/internal/realtime"
)

// Publisher is the slice of realtime.Service that the chat transport needs to
// fan out message + receipt events. Defined here so the chat handler does not
// pull in the entire realtime service surface.
type Publisher interface {
	PublishMessageCreated(ctx context.Context, topic string, message any) error
	PublishMessageRead(ctx context.Context, topic string, receipt any) error
}

type Handler struct {
	svc       *Service
	publisher Publisher
}

func NewHandler(svc *Service, publisher Publisher) *Handler {
	return &Handler{svc: svc, publisher: publisher}
}

func (h *Handler) RegisterRoutes(_ *gin.RouterGroup, protected *gin.RouterGroup) {
	protected.POST("/direct-chats", h.createOrGetDirect)
	protected.GET("/direct-chats/:direct_chat_id/messages", h.listDirectMessages)
	protected.POST("/direct-chats/:direct_chat_id/messages", h.sendDirectMessage)
	protected.POST("/direct-chats/:direct_chat_id/read", h.markDirectRead)

	protected.GET("/channels/:channel_id/messages", h.listChannelMessages)
	protected.POST("/channels/:channel_id/messages", h.sendChannelMessage)
	protected.POST("/channels/:channel_id/read", h.markChannelRead)
}

func (h *Handler) createOrGetDirect(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	var req CreateDirectChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	targetID, err := httpapi.ParseUUIDParam(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	chatRec, err := h.svc.CreateOrGetDirectChat(c.Request.Context(), CreateDirectChatInput{
		UserID: userID, TargetUserID: targetID,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDirectChatResponse(chatRec))
}

func (h *Handler) sendChannelMessage(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	channelID, err := httpapi.ParseUUIDParam(c.Param("channel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req SendMessageRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	msg, err := h.svc.SendChannelMessage(c.Request.Context(), SendChannelMessageInput{
		ChannelID: channelID, UserID: userID, Content: req.Content,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	resp := toMessageResponse(msg)
	if h.publisher != nil {
		_ = h.publisher.PublishMessageCreated(c.Request.Context(), realtime.ChannelTopic(channelID), resp)
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) listChannelMessages(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	channelID, err := httpapi.ParseUUIDParam(c.Param("channel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	limit, offset, err := httpapi.ParsePaging(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	messages, err := h.svc.ListChannelMessages(c.Request.Context(), channelID, userID, ListInput{Limit: limit, Offset: offset})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": toMessagesResponse(messages)})
}

func (h *Handler) markChannelRead(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	channelID, err := httpapi.ParseUUIDParam(c.Param("channel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req MarkMessageReadRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	messageID, err := httpapi.ParseUUIDParam(req.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	receipt, err := h.svc.MarkChannelMessageRead(c.Request.Context(), MarkChannelReadInput{
		ChannelID: channelID, UserID: userID, MessageID: messageID,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	resp := toReceiptResponse(receipt)
	if h.publisher != nil {
		_ = h.publisher.PublishMessageRead(c.Request.Context(), realtime.ChannelTopic(channelID), resp)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) sendDirectMessage(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	directID, err := httpapi.ParseUUIDParam(c.Param("direct_chat_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req SendMessageRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	msg, err := h.svc.SendDirectMessage(c.Request.Context(), SendDirectMessageInput{
		DirectChatID: directID, UserID: userID, Content: req.Content,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	resp := toMessageResponse(msg)
	if h.publisher != nil {
		_ = h.publisher.PublishMessageCreated(c.Request.Context(), realtime.DirectTopic(directID), resp)
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) listDirectMessages(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	directID, err := httpapi.ParseUUIDParam(c.Param("direct_chat_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	limit, offset, err := httpapi.ParsePaging(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	messages, err := h.svc.ListDirectMessages(c.Request.Context(), directID, userID, ListInput{Limit: limit, Offset: offset})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": toMessagesResponse(messages)})
}

func (h *Handler) markDirectRead(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	directID, err := httpapi.ParseUUIDParam(c.Param("direct_chat_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	var req MarkMessageReadRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	messageID, err := httpapi.ParseUUIDParam(req.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	receipt, err := h.svc.MarkDirectMessageRead(c.Request.Context(), MarkDirectReadInput{
		DirectChatID: directID, UserID: userID, MessageID: messageID,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}
	resp := toReceiptResponse(receipt)
	if h.publisher != nil {
		_ = h.publisher.PublishMessageRead(c.Request.Context(), realtime.DirectTopic(directID), resp)
	}
	c.JSON(http.StatusOK, resp)
}
