package handlers

import (
	"github.com/jsiebens/ionscale/pkg/sdk/dnsplugin"
	"github.com/labstack/echo/v4"
	"github.com/libdns/libdns"
	"net"
	"net/http"
	"strings"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"time"
)

func NewDNSHandlers(_ key.MachinePublic, zone string, provider dnsplugin.Provider) *DNSHandlers {
	return &DNSHandlers{
		zone:     zone,
		provider: provider,
	}
}

type DNSHandlers struct {
	zone     string
	provider dnsplugin.Provider
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

	_, err := h.provider.SetRecords(ctx, h.zone, []libdns.Record{
		libdns.RR{
			Type: req.Type,
			Name: libdns.RelativeName(req.Name, h.zone),
			Data: req.Value,
			TTL:  1 * time.Minute,
		}})

	if err != nil {
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
				records, _ := net.LookupTXT(req.Name)
				for _, txt := range records {
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
