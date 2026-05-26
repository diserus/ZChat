package realtime

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zchat/internal/httpapi"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(_ *gin.RouterGroup, protected *gin.RouterGroup) {
	protected.GET("/presence/:user_id", h.getPresence)
}

// GetPresence godoc
// @Summary      Get user presence status (from Redis)
// @Tags         presence
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Success      200  {object}  map[string]interface{}  "{\"user_id\":\"...\",\"status\":\"online\",\"last_seen\":\"...\"}"
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /presence/{user_id} [get]
func (h *Handler) getPresence(c *gin.Context) {
	userID, err := httpapi.ParseUUIDParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	presence, err := h.svc.GetPresence(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httpapi.ErrorJSON(err))
		return
	}
	c.JSON(http.StatusOK, presence)
}
