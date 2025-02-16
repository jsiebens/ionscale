package handlers

import (
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

func NewUpdateHealthHandlers(machineKey key.MachinePublic, repository domain.Repository) *UpdateFeatureHandlers {
	return &UpdateFeatureHandlers{machineKey: machineKey, repository: repository}
}

type UpdateFeatureHandlers struct {
	machineKey key.MachinePublic
	repository domain.Repository
}

func (h *UpdateFeatureHandlers) UpdateHealth(c echo.Context) error {
	ctx := c.Request().Context()

	req := new(tailcfg.HealthChangeRequest)
	if err := c.Bind(req); err != nil {
		return logError(err)
	}

	machineKey := h.machineKey.String()
	nodeKey := req.NodeKey.String()

	machine, err := h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
	if err != nil {
		return err
	}

	if machine == nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	zap.L().Debug("Health checks updated",
		zap.Uint64("tailnet", machine.TailnetID),
		zap.Uint64("machine", machine.ID),
		zap.String("subsystem", req.Subsys),
		zap.String("err", req.Error),
	)

	return c.String(http.StatusOK, "OK")
}
