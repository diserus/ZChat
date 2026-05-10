package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"zchat/internal/httpapi"
)

type contextKey string

const (
	ctxUserID contextKey = "user_id"
	ctxEmail  contextKey = "email"
)

// AccessTokenValidator is a narrow interface to allow tests to stub the JWT layer.
type AccessTokenValidator interface {
	ValidateAccessToken(tokenStr string) (*Claims, error)
}

func Middleware(jwt AccessTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			if token := c.Query("token"); token != "" {
				if !applyClaims(c, jwt, token) {
					return
				}
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpapi.ErrorResponse{Error: "authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpapi.ErrorResponse{Error: "invalid authorization format"})
			return
		}
		if !applyClaims(c, jwt, parts[1]) {
			return
		}
		c.Next()
	}
}

func applyClaims(c *gin.Context, jwt AccessTokenValidator, token string) bool {
	claims, err := jwt.ValidateAccessToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, httpapi.ErrorResponse{Error: "invalid or expired token"})
		return false
	}
	ctx := context.WithValue(c.Request.Context(), ctxUserID, claims.UserID)
	ctx = context.WithValue(ctx, ctxEmail, claims.Email)
	c.Request = c.Request.WithContext(ctx)
	return true
}

func UserIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	v := ctx.Value(ctxUserID)
	if v == nil {
		return uuid.Nil, fmt.Errorf("user_id not found in context")
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user_id has wrong type")
	}
	return id, nil
}

func EmailFromCtx(ctx context.Context) (string, error) {
	v := ctx.Value(ctxEmail)
	if v == nil {
		return "", fmt.Errorf("email not found in context")
	}
	email, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("email has wrong type")
	}
	return email, nil
}

// RequireUserID resolves the authenticated user from request context or aborts
// the request with 401. Returns the user ID and a boolean indicating whether
// the handler should proceed.
func RequireUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := UserIDFromCtx(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, httpapi.ErrorJSON(err))
		return uuid.Nil, false
	}
	return id, true
}

// ErrNoUser is returned when handler-side context lookup fails — exported so
// that other packages can identify it if they propagate the error.
var ErrNoUser = errors.New("no authenticated user")
