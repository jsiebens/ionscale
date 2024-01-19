package handlers

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

func NewQueryFeatureHandlers(machineKey key.MachinePublic, dnsProvider dns.Provider, repository domain.Repository) *QueryFeatureHandlers {
	return &QueryFeatureHandlers{
		machineKey:  machineKey,
		dnsProvider: dnsProvider,
		repository:  repository,
	}
}

type QueryFeatureHandlers struct {
	machineKey  key.MachinePublic
	dnsProvider dns.Provider
	repository  domain.Repository
}

func (h *QueryFeatureHandlers) QueryFeature(c echo.Context) error {
	ctx := c.Request().Context()

	req := new(tailcfg.QueryFeatureRequest)
	if err := c.Bind(req); err != nil {
		return logError(err)
	}

	machineKey := h.machineKey.String()
	nodeKey := req.NodeKey.String()

	resp := tailcfg.QueryFeatureResponse{Complete: true}

	switch req.Feature {
	case "serve":
		machine, err := h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
		if err != nil {
			return err
		}

		if machine == nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		if h.dnsProvider == nil || !machine.Tailnet.DNSConfig.HttpsCertsEnabled {
			resp.Text = fmt.Sprintf(serverMessage, machine.Tailnet.Name)
			resp.Complete = false
		}
	case "funnel":
		resp.Text = fmt.Sprintf("Sorry, ionscale has no support for feature '%s'\n", req.Feature)
		resp.Complete = false
	default:
		resp.Text = fmt.Sprintf("Unknown feature request '%s'\n", req.Feature)
		resp.Complete = false
	}

	return c.JSON(http.StatusOK, resp)
}

const serverMessage = `Enabling HTTPS is required to use Serve:

  ionscale tailnets set-dns --tailnet %s --https-certs=true --magic-dns
`
