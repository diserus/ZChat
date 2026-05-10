package app

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"zchat/internal/auth"
	"zchat/internal/httpapi"
)

// routeRegistrar is the contract every bounded-context HTTP handler implements.
type routeRegistrar interface {
	RegisterRoutes(public, protected *gin.RouterGroup)
}

func newRouter(log *zap.Logger, jwt auth.AccessTokenValidator, registrars []routeRegistrar) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), httpapi.RequestLogger(log))

	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	v1 := router.Group("/api/v1")
	public := v1.Group("/")
	protected := v1.Group("/")
	protected.Use(auth.Middleware(jwt))

	for _, r := range registrars {
		r.RegisterRoutes(public, protected)
	}
	return router
}
