package voice

import (
	"errors"
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
	protected.GET("/groups/:group_id/channels/:channel_id/voice-moderation-events", h.listEvents)
}

// GetVoiceModerationEvents godoc
// @Summary      Get voice moderation history for a voice channel
// @Tags         channels
// @Security     BearerAuth
// @Param        group_id path string true "Group ID"
// @Param        channel_id path string true "Voice Channel ID"
// @Param        limit query int false "Events per page" default(50)
// @Param        cursor query string false "Pagination cursor"
// @Param        from query string false "Start time (RFC3339)" example("2023-01-01T00:00:00Z")
// @Param        to query string false "End time (RFC3339)"
// @Success      200  {object}  voice.ListEventsResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Failure      403  {object}  httpapi.ErrorResponse  "Not voice channel"
// @Router       /groups/{group_id}/channels/{channel_id}/voice-moderation-events [get]
func (h *Handler) listEvents(c *gin.Context) {
	userID, ok := auth.RequireUserID(c)
	if !ok {
		return
	}
	groupID, err := httpapi.ParseUUIDParam(c.Param("group_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	channelID, err := httpapi.ParseUUIDParam(c.Param("channel_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		v, parseErr := httpapi.ParsePositiveInt(raw)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(errors.New("invalid limit")))
			return
		}
		limit = v
	}
	from, err := httpapi.ParseOptionalRFC3339(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	to, err := httpapi.ParseOptionalRFC3339(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	cursor, err := ParseCursor(c.Query("cursor"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}

	events, err := h.svc.ListEvents(c.Request.Context(), ListEventsInput{
		GroupID:   groupID,
		ChannelID: channelID,
		UserID:    userID,
		From:      from,
		To:        to,
		Cursor:    cursor,
		Limit:     limit,
	})
	if err != nil {
		httpapi.RespondError(c, err)
		return
	}

	var nextCursor string
	if len(events) == limit {
		last := events[len(events)-1]
		nextCursor = FormatCursor(Cursor{CreatedAt: last.CreatedAt.UTC(), EventID: last.ID})
	}

	c.JSON(http.StatusOK, ListEventsResponse{
		Events:     toEventsResponse(events),
		NextCursor: nextCursor,
	})
}
