package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"tailscale.com/types/opt"
	"time"
)

func NewPollNetMapHandler(
	createBinder bind.Factory,
	brokers broker.Pubsub,
	repository domain.Repository,
	offlineTimers *OfflineTimers) *PollNetMapHandler {

	handler := &PollNetMapHandler{
		createBinder:  createBinder,
		brokers:       brokers,
		repository:    repository,
		offlineTimers: offlineTimers,
	}

	return handler
}

type PollNetMapHandler struct {
	createBinder  bind.Factory
	repository    domain.Repository
	brokers       broker.Pubsub
	offlineTimers *OfflineTimers
}

func (h *PollNetMapHandler) PollNetMap(c echo.Context) error {
	ctx := c.Request().Context()
	binder, err := h.createBinder(c)
	if err != nil {
		return err
	}

	req := &tailcfg.MapRequest{}
	if err := binder.BindRequest(c, req); err != nil {
		return err
	}

	machineKey := binder.Peer().String()
	nodeKey := req.NodeKey.String()

	var m *domain.Machine
	m, err = h.repository.GetMachineByKeys(ctx, machineKey, nodeKey)
	if err != nil {
		return err
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
		return err
	}

	tailnetID := m.TailnetID
	machineID := m.ID

	h.brokers.Publish(tailnetID, &broker.Signal{PeerUpdated: &machineID})

	if !mapRequest.Stream {
		return c.String(http.StatusOK, "")
	}

	var syncedPeers = make(map[uint64]bool)
	var derpMapChecksum = ""

	response, syncedPeers, derpMapChecksum, err := h.createMapResponse(m, binder, mapRequest, false, make(map[uint64]bool), derpMapChecksum)
	if err != nil {
		return err
	}

	updateChan := make(chan *broker.Signal, 20)

	unsubscribe, err := h.brokers.Subscribe(tailnetID, updateChan)
	if err != nil {
		return err
	}
	h.cancelOfflineMessage(machineID)

	// Listen to connection close
	notify := c.Request().Context().Done()

	keepAliveResponse, err := h.createKeepAliveResponse(binder, mapRequest)
	if err != nil {
		return err
	}
	keepAliveTicker := time.NewTicker(config.KeepAliveInterval())
	syncTicker := time.NewTicker(5 * time.Second)

	c.Response().WriteHeader(http.StatusOK)

	if _, err := c.Response().Write(response); err != nil {
		return err
	}
	c.Response().Flush()

	connectedDevices.WithLabelValues(m.Tailnet.Name).Inc()

	defer func() {
		connectedDevices.WithLabelValues(m.Tailnet.Name).Dec()
		unsubscribe()
		keepAliveTicker.Stop()
		syncTicker.Stop()
		_ = h.repository.SetMachineLastSeen(ctx, machineID)
		h.scheduleOfflineMessage(tailnetID, machineID)
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
					return err
				}
				_ = h.repository.SetMachineLastSeen(ctx, machineID)
				c.Response().Flush()
			}
		case <-syncTicker.C:
			if latestSync.Before(latestUpdate) {
				machine, err := h.repository.GetMachine(ctx, machineID)
				if err != nil {
					return err
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
					return err
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
		return err
	}

	response, _, _, err := h.createMapResponse(m, binder, request, false, map[uint64]bool{}, "")
	if err != nil {
		return err
	}

	_, err = c.Response().Write(response)
	return err
}

func (h *PollNetMapHandler) scheduleOfflineMessage(tailnetID, machineID uint64) {
	h.offlineTimers.startCh <- [2]uint64{tailnetID, machineID}
}

func (h *PollNetMapHandler) cancelOfflineMessage(machineID uint64) {
	h.offlineTimers.stopCh <- machineID
}

func (h *PollNetMapHandler) createKeepAliveResponse(binder bind.Binder, request *tailcfg.MapRequest) ([]byte, error) {
	mapResponse := &tailcfg.MapResponse{
		KeepAlive: true,
	}

	return binder.Marshal(request.Compress, mapResponse)
}

func (h *PollNetMapHandler) createMapResponse(m *domain.Machine, binder bind.Binder, request *tailcfg.MapRequest, delta bool, prevSyncedPeerIDs map[uint64]bool, prevDerpMapChecksum string) ([]byte, map[uint64]bool, string, error) {
	ctx := context.TODO()

	tailnet, err := h.repository.GetTailnet(ctx, m.TailnetID)
	if err != nil {
		return nil, nil, "", err
	}

	node, user, err := mapping.ToNode(m, tailnet)
	if err != nil {
		return nil, nil, "", err
	}

	policies := tailnet.ACLPolicy
	var users = []tailcfg.UserProfile{*user}
	var changedPeers []*tailcfg.Node
	var removedPeers []tailcfg.NodeID
	var validPeers []domain.Machine

	candidatePeers, err := h.repository.ListMachinePeers(ctx, m.TailnetID, m.MachineKey)
	if err != nil {
		return nil, nil, "", err
	}

	syncedPeerIDs := map[uint64]bool{}
	syncedUserIDs := map[tailcfg.UserID]bool{}

	for _, peer := range candidatePeers {
		if peer.IsExpired() {
			continue
		}
		if policies.IsValidPeer(m, &peer) || policies.IsValidPeer(&peer, m) {
			validPeers = append(validPeers, peer)
			n, u, err := mapping.ToNode(&peer, tailnet)
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

	rules := policies.BuildFilterRules(candidatePeers, m)

	controlTime := time.Now().UTC()
	var mapResponse *tailcfg.MapResponse

	if !delta {
		mapResponse = &tailcfg.MapResponse{
			KeepAlive:       false,
			Node:            node,
			DNSConfig:       mapping.ToDNSConfig(m, validPeers, &m.Tailnet, &dnsConfig),
			PacketFilter:    rules,
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
			DNSConfig:       mapping.ToDNSConfig(m, validPeers, &m.Tailnet, &dnsConfig),
			PacketFilter:    rules,
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

	if request.OmitPeers {
		mapResponse.PeersChanged = nil
		mapResponse.PeersRemoved = nil
		mapResponse.Peers = nil
	}

	payload, err := binder.Marshal(request.Compress, mapResponse)

	return payload, syncedPeerIDs, derpMap.Checksum, nil
}

func NewOfflineTimers(repository domain.Repository, pubsub broker.Pubsub) *OfflineTimers {
	return &OfflineTimers{
		repository: repository,
		pubsub:     pubsub,
		data:       make(map[uint64]*time.Timer),
		startCh:    make(chan [2]uint64),
		stopCh:     make(chan uint64),
	}
}

type OfflineTimers struct {
	repository domain.Repository
	pubsub     broker.Pubsub
	data       map[uint64]*time.Timer
	stopCh     chan uint64
	startCh    chan [2]uint64
}

func (o *OfflineTimers) Start() {
	for {
		select {
		case i := <-o.startCh:
			o.scheduleOfflineMessage(i[0], i[1])
		case m := <-o.stopCh:
			o.cancelOfflineMessage(m)
		}
	}
}

func (o *OfflineTimers) scheduleOfflineMessage(tailnetID, machineID uint64) {
	t, ok := o.data[machineID]
	if ok {
		t.Stop()
		delete(o.data, machineID)
	}

	timer := time.NewTimer(config.KeepAliveInterval())
	go func() {
		<-timer.C
		o.pubsub.Publish(tailnetID, &broker.Signal{PeerUpdated: &machineID})
		o.stopCh <- machineID
	}()

	o.data[machineID] = timer
}

func (o *OfflineTimers) cancelOfflineMessage(machineID uint64) {
	t, ok := o.data[machineID]
	if ok {
		t.Stop()
		delete(o.data, machineID)
	}
}

func optBool(v bool) opt.Bool {
	b := opt.Bool("")
	b.Set(v)
	return b
}
