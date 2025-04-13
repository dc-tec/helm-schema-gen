package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dc-tec/helm-schema-gen/cmd/helm-schema-gen/tool"
	"github.com/dc-tec/helm-schema-gen/pkg/logging"
)

// Entrypoint for the helm-schema-gen tool
func run() error {
	// Create a context that can be canceled on signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-signalChan:
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	// Initialize the logger and add application component
	logger := logging.WithComponent(ctx, "application")
	logger.InfoContext(ctx, "Starting helm-schema-gen")

	// Run the schema generator
	return tool.GenerateSchema(ctx)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
