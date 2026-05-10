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
