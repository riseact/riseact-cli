package services

import (
	"context"
	"os"
	"os/signal"
	"riseact/internal/utils/logger"
	"strconv"
	"syscall"
)

// ProxyApp exposes an already-running local server, without starting anything
// itself.
func ProxyApp(port string) error {
	logger.Debug("Starting proxy on port " + port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	localPort, err := strconv.Atoi(port)

	if err != nil {
		return err
	}

	_, access, err := startDevAccess(localPort)

	if err != nil {
		return err
	}

	defer access.Close()

	logger.Info("")
	logger.Infof("App url: %s", access.URL)
	logger.Info("")

	<-ctx.Done()

	logger.Info("Stopping the tunnel...")

	return nil
}
