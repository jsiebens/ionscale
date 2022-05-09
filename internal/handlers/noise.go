package handlers

import (
	"context"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net/http"
	"tailscale.com/control/controlhttp"
	"tailscale.com/net/netutil"
	"tailscale.com/types/key"
)

type NoiseHandlers struct {
	controlKey        key.MachinePrivate
	createPeerHandler CreatePeerHandler
}

type CreatePeerHandler func(p key.MachinePublic) http.Handler

func NewNoiseHandlers(controlKey key.MachinePrivate, createPeerHandler CreatePeerHandler) *NoiseHandlers {
	return &NoiseHandlers{
		controlKey:        controlKey,
		createPeerHandler: createPeerHandler,
	}
}

func (h *NoiseHandlers) Upgrade(c echo.Context) error {
	conn, err := controlhttp.AcceptHTTP(context.Background(), c.Response(), c.Request(), h.controlKey)
	if err != nil {
		return err
	}

	handler := h.createPeerHandler(conn.Peer())

	server := http.Server{}
	server.Handler = h2c.NewHandler(handler, &http2.Server{})
	return server.Serve(netutil.NewOneConnListener(conn, nil))
}
