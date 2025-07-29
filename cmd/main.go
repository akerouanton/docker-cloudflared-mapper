package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/akerouanton/docker-cloudflared-mapper/portmapper"
	"github.com/cloudflare/cloudflare-go/v4"
)

func main() {
	accountId := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	if accountId == "" {
		slog.Error("CLOUDFLARE_ACCOUNT_ID is not set")
		os.Exit(1)
	}

	plugin := portmapper.NewPlugin(cloudflare.NewClient(), accountId)

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer sigCancel()

	if err := plugin.ListenAndServe(sigCtx); err != nil {
		slog.Error("failed to listen and serve", "error", err)
		os.Exit(1)
	}
}
