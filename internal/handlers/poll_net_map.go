package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"time"
)

func NewPollNetMapHandler(
	createBinder bind.Factory,
	sessionManager core.PollMapSessionManager,
	repository domain.Repository) *PollNetMapHandler {

	handler := &PollNetMapHandler{
		createBinder:   createBinder,
		sessionManager: sessionManager,
		repository:     repository,
	}

	return handler
}

type PollNetMapHandler struct {
	createBinder   bind.Factory
	repository     domain.Repository
	sessionManager core.PollMapSessionManager
}

func (h *PollNetMapHandler) PollNetMap(c echo.Context) error {
	ctx := c.Request().Context()
	binder, err := h.createBinder(c)
	if err != nil {
		return logError(err)
	}

	req := &tailcfg.MapRequest{}
	if err := binder.BindRequest(c, req); err != nil {
		return logError(err)
	}

	machineKey := binder.Peer().String()
	nodeKey := req.NodeKey.String()

	var m *domain.Machine
	m, err = h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
	if err != nil {
		return logError(err)
	}

	if m == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if req.ReadOnly {
		return h.handleReadOnly(c, binder, m, req)
	} else {
		return h.handleUpdate(c, binder, m, req)
	}
}

func (h *PollNetMapHandler) handleUpdate(c echo.Context, binder bind.Binder, m *domain.Machine, mapRequest *tailcfg.MapRequest) error {
	ctx := c.Request().Context()

	now := time.Now().UTC()

	m.HostInfo = domain.HostInfo(*mapRequest.Hostinfo)
	m.DiscoKey = mapRequest.DiscoKey.String()
	m.Endpoints = mapRequest.Endpoints
	m.LastSeen = &now

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return logError(err)
	}

	tailnetID := m.TailnetID
	machineID := m.ID

	h.sessionManager.NotifyAll(tailnetID, m.ID)

	if !mapRequest.Stream {
		return c.String(http.StatusOK, "")
	}

	mapper := mapping.NewPollNetMapper(mapRequest, m.ID, h.repository, h.sessionManager)

	response, err := createMapResponse(mapper, binder, false, mapRequest.Compress)
	if err != nil {
		return logError(err)
	}

	updateChan := make(chan *core.Ping, 20)
	h.sessionManager.Register(m.TailnetID, m.ID, updateChan)

	// Listen to connection close
	notify := c.Request().Context().Done()

	keepAliveResponse, err := createKeepAliveResponse(binder, mapRequest)
	if err != nil {
		return logError(err)
	}

	c.Response().WriteHeader(http.StatusOK)

	if _, err := c.Response().Write(response); err != nil {
		return logError(err)
	}
	c.Response().Flush()

	connectedDevices.WithLabelValues(m.Tailnet.Name).Inc()

	keepAliveTicker := time.NewTicker(config.KeepAliveInterval())
	syncTicker := time.NewTicker(5 * time.Second)

	defer func() {
		connectedDevices.WithLabelValues(m.Tailnet.Name).Dec()
		h.sessionManager.Deregister(m.TailnetID, m.ID)
		keepAliveTicker.Stop()
		syncTicker.Stop()
		_ = h.repository.SetMachineLastSeen(ctx, machineID)
	}()

	var latestSync = time.Now()
	var latestUpdate = latestSync

	for {
		select {
		case <-updateChan:
			latestUpdate = time.Now()
		case <-keepAliveTicker.C:
			if mapRequest.KeepAlive {
				if _, err := c.Response().Write(keepAliveResponse); err != nil {
					return logError(err)
				}
				_ = h.repository.SetMachineLastSeen(ctx, machineID)
				c.Response().Flush()
			}
		case <-syncTicker.C:
			if latestSync.Before(latestUpdate) {
				machine, err := h.repository.GetMachine(ctx, machineID)
				if err != nil {
					return logError(err)
				}
				if machine == nil {
					return nil
				}

				var payload []byte
				var payloadErr error

				payload, payloadErr = createMapResponse(mapper, binder, true, mapRequest.Compress)

				if payloadErr != nil {
					return payloadErr
				}

				if _, err := c.Response().Write(payload); err != nil {
					return logError(err)
				}
				c.Response().Flush()

				latestSync = latestUpdate
			}
		case <-notify:
			return nil
		}
	}
}

func (h *PollNetMapHandler) handleReadOnly(c echo.Context, binder bind.Binder, m *domain.Machine, request *tailcfg.MapRequest) error {
	ctx := c.Request().Context()

	m.HostInfo = domain.HostInfo(*request.Hostinfo)
	m.DiscoKey = request.DiscoKey.String()

	if err := h.repository.SaveMachine(ctx, m); err != nil {
		return logError(err)
	}

	mapper := mapping.NewPollNetMapper(request, m.ID, h.repository, h.sessionManager)
	payload, err := createMapResponse(mapper, binder, false, request.Compress)
	if err != nil {
		return logError(err)
	}

	_, err = c.Response().Write(payload)
	return logError(err)
}

func createKeepAliveResponse(binder bind.Binder, request *tailcfg.MapRequest) ([]byte, error) {
	mapResponse := &tailcfg.MapResponse{
		KeepAlive: true,
	}

	return binder.Marshal(request.Compress, mapResponse)
}

func createMapResponse(m *mapping.PollNetMapper, binder bind.Binder, delta bool, compress string) ([]byte, error) {
	response, err := m.CreateMapResponse(context.Background(), delta)
	if err != nil {
		return nil, err
	}
	return binder.Marshal(compress, response)
}
