package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Power-Monitor/internal/cmd"
	"Power-Monitor/internal/kernel"
	"Power-Monitor/model"
	"Power-Monitor/router"
	"Power-Monitor/settings"

	"github.com/gin-gonic/gin"
	"github.com/uozi-tech/cosy"
	cKernel "github.com/uozi-tech/cosy/kernel"
	"github.com/uozi-tech/cosy/logger"
	cRouter "github.com/uozi-tech/cosy/router"
	cSettings "github.com/uozi-tech/cosy/settings"
)

func main() {
	appCmd := cmd.NewAppCmd()

	confPath := appCmd.String("config")
	settings.Init(confPath)

	// Set gin mode
	gin.SetMode(cSettings.ServerSettings.RunMode)

	// Initialize logger
	logger.Init(cSettings.ServerSettings.RunMode)
	defer logger.Sync()
	defer logger.Info("Power Monitor Server exited")

	// Initialize cosy
	// Register models
	cosy.RegisterModels(model.GenerateAllModel()...)

	// Register initialization functions
	cosy.RegisterInitFunc(func() {
		ctx := context.Background()
		kernel.Boot(ctx)
		router.InitRouter()
	})

	// Initialize router
	cRouter.Init()

	// Boot kernel
	ctx := context.Background()
	cKernel.Boot(ctx)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cSettings.ServerSettings.Host, cSettings.ServerSettings.Port),
		Handler: cRouter.GetEngine(),
		// Set timeouts
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting Power Monitor server on %s", srv.Addr)

		var err error
		if cSettings.ServerSettings.EnableHTTPS {
			// TLS configuration for HTTPS
			srv.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
			logger.Info("Starting Power Monitor HTTPS server")
			err = srv.ListenAndServeTLS("", "")
		} else {
			logger.Info("Starting Power Monitor HTTP server")
			err = srv.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()

	// Graceful shutdown
	logger.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}
