package handlers

import (
	stderrors "errors"
	"github.com/jsiebens/ionscale/internal/errors"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
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
	conn, err := controlhttp.AcceptHTTP(c.Request().Context(), c.Response(), c.Request(), h.controlKey)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	handler := h.createPeerHandler(conn.Peer())

	server := http.Server{}
	server.Handler = h2c.NewHandler(handler, &http2.Server{})
	if err := server.Serve(netutil.NewOneConnListener(conn, nil)); err != nil && !stderrors.Is(err, io.EOF) {
		return err
	}
	return nil
}
