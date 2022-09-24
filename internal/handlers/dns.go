package handlers

import (
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
	"strings"
	"tailscale.com/tailcfg"
	"time"
)

func NewDNSHandlers(createBinder bind.Factory, provider dns.Provider) *DNSHandlers {
	return &DNSHandlers{
		createBinder: createBinder,
		provider:     provider,
	}
}

type DNSHandlers struct {
	createBinder bind.Factory
	provider     dns.Provider
}

func (h *DNSHandlers) SetDNS(c echo.Context) error {
	ctx := c.Request().Context()

	binder, err := h.createBinder(c)
	if err != nil {
		return err
	}

	req := &tailcfg.SetDNSRequest{}
	if err := binder.BindRequest(c, req); err != nil {
		return err
	}

	if h.provider == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := h.provider.SetRecord(ctx, req.Type, req.Name, req.Value); err != nil {
		return err
	}

	if strings.HasPrefix(req.Name, "_acme-challenge") && req.Type == "TXT" {
		// Listen to connection close
		notify := ctx.Done()
		timeout := time.After(5 * time.Minute)
		tick := time.NewTicker(5 * time.Second)

		defer func() { tick.Stop() }()

		for {
			select {
			case <-tick.C:
				txtrecords, _ := net.LookupTXT(req.Name)
				for _, txt := range txtrecords {
					if txt == req.Value {
						return binder.WriteResponse(c, http.StatusOK, tailcfg.SetDNSResponse{})
					}
				}
			case <-timeout:
				return binder.WriteResponse(c, http.StatusOK, tailcfg.SetDNSResponse{})
			case <-notify:
				return nil
			}
		}
	}

	return binder.WriteResponse(c, http.StatusOK, tailcfg.SetDNSResponse{})
}
