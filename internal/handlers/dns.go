package handlers

import (
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
	"strings"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"time"
)

func NewDNSHandlers(_ key.MachinePublic, provider dns.Provider) *DNSHandlers {
	return &DNSHandlers{
		provider: provider,
	}
}

type DNSHandlers struct {
	provider dns.Provider
}

func (h *DNSHandlers) SetDNS(c echo.Context) error {
	ctx := c.Request().Context()

	req := &tailcfg.SetDNSRequest{}
	if err := c.Bind(req); err != nil {
		return logError(err)
	}

	if req.Version < SupportedCapabilityVersion {
		return UnsupportedClientVersionError
	}

	if h.provider == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := h.provider.SetRecord(ctx, req.Type, req.Name, req.Value); err != nil {
		return logError(err)
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
						return c.JSON(http.StatusOK, tailcfg.SetDNSResponse{})
					}
				}
			case <-timeout:
				return c.JSON(http.StatusOK, tailcfg.SetDNSResponse{})
			case <-notify:
				return nil
			}
		}
	}

	return c.JSON(http.StatusOK, tailcfg.SetDNSResponse{})
}
