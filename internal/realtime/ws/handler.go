package ws

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"zchat/internal/auth"
	"zchat/internal/realtime"
)

type Handler struct {
	groups   GroupService
	chats    ChatService
	voice    VoiceAudit
	bus      RealtimeBus
	upgrader websocket.Upgrader
}

func NewHandler(groups GroupService, chats ChatService, voice VoiceAudit, bus RealtimeBus) *Handler {
	return &Handler{
		groups:   groups,
		chats:    chats,
		voice:    voice,
		bus:      bus,
		upgrader: websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }},
	}
}

func (h *Handler) RegisterRoutes(_ *gin.RouterGroup, protected *gin.RouterGroup) {
	protected.GET("/ws", h.handle)
}

func (h *Handler) handle(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.bus.MarkOnline(c.Request.Context(), userID)
	h.broadcastPresence(c.Request.Context(), userID, "online")

	wsCtx, cancel := context.WithCancel(context.Background())
	cl := &client{
		userID:    userID,
		conn:      conn,
		send:      make(chan []byte, 128),
		subs:      map[string]func(){},
		voiceSubs: map[uuid.UUID]struct{}{},
		mu:        &sync.Mutex{},
		groups:    h.groups,
		chats:     h.chats,
		voice:     h.voice,
		bus:       h.bus,
		ctx:       wsCtx,
	}

	go cl.writePump()
	cl.readPump()

	cancel()
	cl.close()
	_ = h.bus.MarkOffline(context.Background(), userID)
	h.broadcastPresence(context.Background(), userID, "offline")
}

func (h *Handler) broadcastPresence(ctx context.Context, userID uuid.UUID, status string) {
	groups, err := h.groups.ListUserGroups(ctx, userID)
	if err != nil {
		return
	}
	presence := realtime.Presence{UserID: userID, Status: status}
	for _, g := range groups {
		_ = h.bus.PublishPresenceChanged(ctx, realtime.GroupPresenceTopic(g.ID), presence)
	}
}
