package handlers

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/labstack/echo/v4"
	"github.com/libdns/libdns"
	"net"
	"net/http"
	"strings"
	"tailscale.com/tailcfg"
	"time"
)

func NewDNSHandlers(
	createBinder bind.Factory,
	provider dns.Provider,
	zone string) *DNSHandlers {
	return &DNSHandlers{
		createBinder: createBinder,
		provider:     provider,
		zone:         zone,
	}
}

type DNSHandlers struct {
	createBinder bind.Factory
	provider     dns.Provider
	zone         string
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

	name := strings.TrimSuffix(req.Name, h.zone)

	_, err = h.provider.SetRecords(ctx, fmt.Sprintf("%s.", h.zone), []libdns.Record{{
		Type:  req.Type,
		Name:  name,
		Value: req.Value,
		TTL:   0,
	}})
	if err != nil {
		return err
	}

	if strings.HasPrefix(name, "_acme-challenge") && req.Type == "TXT" {
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
