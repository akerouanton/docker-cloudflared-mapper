package portmapper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/docker/go-plugins-helpers/portmapper"
)

const (
	pluginsDir = "/run/docker/plugins"
	DriverName = "cloudflared"
)

type Plugin struct {
	h *portmapper.Handler
}

func NewPlugin(c *cloudflare.Client, accountId string) *Plugin {
	return &Plugin{
		h: portmapper.NewHandler(newDriver(c, accountId)),
	}
}

func (p Plugin) ListenAndServe(ctx context.Context) error {
	if err := os.MkdirAll(pluginsDir, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("creating plugins directory %s: %w", pluginsDir, err)
	}

	socketPath := pluginsDir + "/" + DriverName + ".sock"
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: socketPath})
	if err != nil {
		return fmt.Errorf("listening on unix socket %s: %w", socketPath, err)
	}

	go func() {
		err := p.h.Serve(ln)
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Warn("serving portmapper plugin: %v", "error", err)
		}
	}()

	<-ctx.Done()

	// Parent context has been cancelled, so create a new context with a timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.h.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := ln.Close(); err != nil {
		return err
	}

	return nil
}
