package auth

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

func (h *Handler) RegisterRoutes(public, _ *gin.RouterGroup) {
	g := public.Group("/auth")
	g.POST("/register", h.register)
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
	g.POST("/logout", h.logout)
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	out, err := h.svc.Register(c.Request.Context(), RegisterInput(req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, httpapi.ErrorJSON(err))
		return
	}
	c.JSON(http.StatusCreated, toAuthResponse(out))
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	out, err := h.svc.Login(c.Request.Context(), LoginInput(req))
	if err != nil {
		c.JSON(http.StatusUnauthorized, httpapi.ErrorJSON(err))
		return
	}
	c.JSON(http.StatusOK, toAuthResponse(out))
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	out, err := h.svc.Refresh(c.Request.Context(), RefreshInput(req))
	if err != nil {
		c.JSON(http.StatusUnauthorized, httpapi.ErrorJSON(err))
		return
	}
	c.JSON(http.StatusOK, toAuthResponse(out))
}

func (h *Handler) logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpapi.ErrorJSON(err))
		return
	}
	if err := h.svc.Logout(c.Request.Context(), RefreshInput(req)); err != nil {
		c.JSON(http.StatusInternalServerError, httpapi.ErrorJSON(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
