package handlers

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/types/key"
)

func NewDERPHandler() *DERPHandlers {
	logger := zap.L().Named("derp")
	return &DERPHandlers{
		s: derp.NewServer(key.NewNode(), func(format string, args ...any) {
			logger.Debug(fmt.Sprintf(format, args...))
		}),
	}
}

type DERPHandlers struct {
	s *derp.Server
}

func (h *DERPHandlers) Handler(c echo.Context) error {
	derphttp.Handler(h.s).ServeHTTP(c.Response(), c.Request())
	return nil
}

func (h *DERPHandlers) LatencyCheck(c echo.Context) error {
	return c.String(http.StatusOK, "")
}

func (h *DERPHandlers) DebugTraffic(c echo.Context) error {
	h.s.ServeDebugTraffic(c.Response(), c.Request())
	return nil
}

func (h *DERPHandlers) DebugCheck(c echo.Context) error {
	if err := h.s.ConsistencyCheck(); err != nil {
		return err
	}

	return c.String(http.StatusOK, "DERP Server ConsistencyCheck okay")
}
