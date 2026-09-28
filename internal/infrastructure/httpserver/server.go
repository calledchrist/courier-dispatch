package httpserver

import (
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/config"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/httpserver/handler"
	"github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/health", handler.Health)

	return router
}
