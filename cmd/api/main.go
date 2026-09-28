package main

import (
	"log/slog"
	"os"

	"github.com/calledchrist/courier-dispatch/internal/infrastructure/config"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/container"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	container, err := container.New()
	if err != nil {
		logger.Error("failed to build dependency container", "error", err)
		os.Exit(1)
	}

	err = container.Invoke(func(router *gin.Engine, cfg *config.Config, logger *slog.Logger) error {
		address := cfg.HTTP.Host + ":" + cfg.HTTP.Port
		logger.Info("starting HTTP server", "address", address)

		return router.Run(address)
	})
	if err != nil {
		logger.Error("application stopped with an error", "error", err)
		os.Exit(1)
	}
}
