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

// Register godoc
// @Summary      User registration
// @Description  Create a new user account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "User info"
// @Success      201  {object}  AuthResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      409  {object}  httpapi.ErrorResponse  "User already exists"
// @Router       /auth/register [post]
func (h *Handler) register(c *gin.Context) {
	var req RegisterRequest
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

// Login godoc
// @Summary      Login
// @Description  Authenticate user and return tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Credentials"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
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

// / Refresh godoc
// @Summary      Refresh access token
// @Description  Obtain new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  httpapi.ErrorResponse
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	var req RefreshRequest
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

// Logout godoc
// @Summary      Logout
// @Description  Invalidate refresh token (client should discard tokens)
// @Tags         auth
// @Security     BearerAuth
// @Success      204  "No Content"
// @Failure      401  {object}  httpapi.ErrorResponse
// @Router       /auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	var req RefreshRequest
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
