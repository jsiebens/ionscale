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
	"net/netip"
	"tailscale.com/tailcfg"
	"tailscale.com/types/opt"
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

	h.sessionManager.NotifyAll(tailnetID)

	if !mapRequest.Stream {
		return c.String(http.StatusOK, "")
	}

	var syncedPeers = make(map[uint64]bool)
	var derpMapChecksum = ""

	response, syncedPeers, derpMapChecksum, err := h.createMapResponse(m, binder, mapRequest, false, make(map[uint64]bool), derpMapChecksum)
	if err != nil {
		return logError(err)
	}

	updateChan := make(chan *core.Ping, 20)
	h.sessionManager.Register(m.TailnetID, m.ID, updateChan)

	// Listen to connection close
	notify := c.Request().Context().Done()

	keepAliveResponse, err := h.createKeepAliveResponse(binder, mapRequest)
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

				payload, syncedPeers, derpMapChecksum, payloadErr = h.createMapResponse(machine, binder, mapRequest, true, syncedPeers, derpMapChecksum)

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

	response, _, _, err := h.createMapResponse(m, binder, request, false, map[uint64]bool{}, "")
	if err != nil {
		return logError(err)
	}

	_, err = c.Response().Write(response)
	return logError(err)
}

func (h *PollNetMapHandler) createKeepAliveResponse(binder bind.Binder, request *tailcfg.MapRequest) ([]byte, error) {
	mapResponse := &tailcfg.MapResponse{
		KeepAlive: true,
	}

	return binder.Marshal(request.Compress, mapResponse)
}

func (h *PollNetMapHandler) createMapResponse(m *domain.Machine, binder bind.Binder, request *tailcfg.MapRequest, delta bool, prevSyncedPeerIDs map[uint64]bool, prevDerpMapChecksum string) ([]byte, map[uint64]bool, string, error) {
	ctx := context.TODO()

	prc := &primaryRoutesCollector{flagged: map[netip.Prefix]bool{}}

	tailnet, err := h.repository.GetTailnet(ctx, m.TailnetID)
	if err != nil {
		return nil, nil, "", err
	}

	serviceUser, _, err := h.repository.GetOrCreateServiceUser(ctx, tailnet)
	if err != nil {
		return nil, nil, "", err
	}

	hostinfo := tailcfg.Hostinfo(m.HostInfo)
	node, user, err := mapping.ToNode(m, tailnet, serviceUser, false, true, prc.filter)
	if err != nil {
		return nil, nil, "", err
	}

	policies := tailnet.ACLPolicy
	var users = []tailcfg.UserProfile{*user}
	var changedPeers []*tailcfg.Node
	var removedPeers []tailcfg.NodeID

	candidatePeers, err := h.repository.ListMachinePeers(ctx, m.TailnetID, m.MachineKey)
	if err != nil {
		return nil, nil, "", err
	}

	syncedPeerIDs := map[uint64]bool{}
	syncedUserIDs := map[tailcfg.UserID]bool{user.ID: true}

	for _, peer := range candidatePeers {
		if peer.IsExpired() {
			continue
		}
		if policies.IsValidPeer(m, &peer) || policies.IsValidPeer(&peer, m) {
			isConnected := h.sessionManager.HasSession(peer.TailnetID, peer.ID)

			n, u, err := mapping.ToNode(&peer, tailnet, serviceUser, true, isConnected, prc.filter)
			if err != nil {
				return nil, nil, "", err
			}
			changedPeers = append(changedPeers, n)
			syncedPeerIDs[peer.ID] = true
			delete(prevSyncedPeerIDs, peer.ID)

			if _, ok := syncedUserIDs[u.ID]; !ok {
				users = append(users, *u)
				syncedUserIDs[u.ID] = true
			}
		}
	}

	for p, _ := range prevSyncedPeerIDs {
		removedPeers = append(removedPeers, tailcfg.NodeID(p))
	}

	dnsConfig := tailnet.DNSConfig

	derpMap, err := m.Tailnet.GetDERPMap(ctx, h.repository)
	if err != nil {
		return nil, nil, "", err
	}

	filterRules := policies.BuildFilterRules(candidatePeers, m)

	controlTime := time.Now().UTC()
	var mapResponse *tailcfg.MapResponse

	if !delta {
		mapResponse = &tailcfg.MapResponse{
			KeepAlive:       false,
			Node:            node,
			DNSConfig:       mapping.ToDNSConfig(m, &m.Tailnet, &dnsConfig),
			PacketFilter:    filterRules,
			DERPMap:         &derpMap.DERPMap,
			Domain:          domain.SanitizeTailnetName(m.Tailnet.Name),
			Peers:           changedPeers,
			UserProfiles:    users,
			ControlTime:     &controlTime,
			CollectServices: optBool(tailnet.ServiceCollectionEnabled),
			Debug: &tailcfg.Debug{
				DisableLogTail: true,
			},
		}
	} else {
		mapResponse = &tailcfg.MapResponse{
			Node:            node,
			DNSConfig:       mapping.ToDNSConfig(m, &m.Tailnet, &dnsConfig),
			PacketFilter:    filterRules,
			Domain:          domain.SanitizeTailnetName(m.Tailnet.Name),
			PeersChanged:    changedPeers,
			PeersRemoved:    removedPeers,
			UserProfiles:    users,
			ControlTime:     &controlTime,
			CollectServices: optBool(tailnet.ServiceCollectionEnabled),
		}

		if prevDerpMapChecksum != derpMap.Checksum {
			mapResponse.DERPMap = &derpMap.DERPMap
		}
	}

	if tailnet.SSHEnabled && hostinfo.TailscaleSSHEnabled() {
		mapResponse.SSHPolicy = policies.BuildSSHPolicy(candidatePeers, m)
	}

	if request.OmitPeers {
		mapResponse.PeersChanged = nil
		mapResponse.PeersRemoved = nil
		mapResponse.Peers = nil
	}

	payload, err := binder.Marshal(request.Compress, mapResponse)

	return payload, syncedPeerIDs, derpMap.Checksum, nil
}

func optBool(v bool) opt.Bool {
	b := opt.Bool("")
	b.Set(v)
	return b
}

type primaryRoutesCollector struct {
	flagged map[netip.Prefix]bool
}

func (p *primaryRoutesCollector) filter(m *domain.Machine) []netip.Prefix {
	var result = []netip.Prefix{}
	for _, r := range m.AllowIPs {
		if _, ok := p.flagged[r]; r.Bits() != 0 && !ok {
			result = append(result, r)
			p.flagged[r] = true
		}
	}
	for _, r := range m.AutoAllowIPs {
		if _, ok := p.flagged[r]; r.Bits() != 0 && !ok {
			result = append(result, r)
			p.flagged[r] = true
		}
	}
	return result
}
