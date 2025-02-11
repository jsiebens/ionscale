package handlers

import (
	stderrors "errors"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"io"
	"net/http"
	"tailscale.com/control/controlhttp/controlhttpserver"
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
	conn, err := controlhttpserver.AcceptHTTP(c.Request().Context(), c.Response(), c.Request(), h.controlKey, nil)
	if err != nil {
		return logError(err)
	}

	handler := h.createPeerHandler(conn.Peer())

	server := http.Server{}
	server.Handler = h2c.NewHandler(handler, &http2.Server{})
	if err := server.Serve(netutil.NewOneConnListener(conn, nil)); err != nil && !stderrors.Is(err, io.EOF) {
		return err
	}
	return nil
}

type JsonBinder struct {
	echo.DefaultBinder
}

func (b JsonBinder) Bind(i interface{}, c echo.Context) error {
	if err := b.BindPathParams(c, i); err != nil {
		return err
	}

	method := c.Request().Method
	if method == http.MethodGet || method == http.MethodDelete || method == http.MethodHead {
		if err := b.BindQueryParams(c, i); err != nil {
			return err
		}
	}

	if c.Request().ContentLength == 0 {
		return nil
	}

	if err := c.Echo().JSONSerializer.Deserialize(c, i); err != nil {
		switch err.(type) {
		case *echo.HTTPError:
			return err
		default:
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	}

	return nil
}
