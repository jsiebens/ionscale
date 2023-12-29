package handlers

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
)

func NewQueryFeatureHandlers(createBinder bind.Factory, dnsProvider dns.Provider, repository domain.Repository) *QueryFeatureHandlers {
	return &QueryFeatureHandlers{
		createBinder: createBinder,
		repository:   repository,
	}
}

type QueryFeatureHandlers struct {
	createBinder bind.Factory
	dnsProvider  dns.Provider
	repository   domain.Repository
}

func (h *QueryFeatureHandlers) QueryFeature(c echo.Context) error {
	ctx := c.Request().Context()

	binder, err := h.createBinder(c)
	if err != nil {
		return logError(err)
	}

	req := new(tailcfg.QueryFeatureRequest)
	if err := binder.BindRequest(c, req); err != nil {
		return logError(err)
	}

	machineKey := binder.Peer().String()
	nodeKey := req.NodeKey.String()

	resp := tailcfg.QueryFeatureResponse{}

	switch req.Feature {
	case "serve":
		machine, err := h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
		if err != nil {
			return err
		}

		if machine == nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		if h.dnsProvider == nil || machine.Tailnet.DNSConfig.HttpsCertsEnabled {
			resp.Text = fmt.Sprintf(serverMessage, machine.Tailnet.Name)
		}
	case "funnel":
		resp.Text = fmt.Sprintf("Sorry, ionscale has no support for feature '%s'\n", req.Feature)
	default:
		resp.Text = fmt.Sprintf("Unknown feature request '%s'\n", req.Feature)
	}

	return binder.WriteResponse(c, http.StatusOK, resp)
}

const serverMessage = `Enabling HTTPS is required to use Serve:

  ionscale tailnets set-dns --tailnet %s --https-certs=true --magic-dns
`
