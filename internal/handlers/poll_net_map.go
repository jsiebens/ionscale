package handlers

import (
	"context"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/labstack/echo/v4"
	"net/http"
	"tailscale.com/tailcfg"
	"tailscale.com/util/dnsname"
	"time"
)

const (
	keepAliveInterval = 1 * time.Minute
)

func NewPollNetMapHandler(
	createBinder bind.Factory,
	brokers *broker.BrokerPool,
	repository domain.Repository,
	offlineTimers *OfflineTimers) *PollNetMapHandler {

	handler := &PollNetMapHandler{
		createBinder:  createBinder,
		brokers:       brokers.Get,
		repository:    repository,
		offlineTimers: offlineTimers,
	}

	return handler
}

type PollNetMapHandler struct {
	createBinder  bind.Factory
	repository    domain.Repository
	brokers       func(uint64) broker.Broker
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

	tailnetBroker := h.brokers(tailnetID)
	tailnetBroker.SignalPeerUpdated(machineID)

	if !mapRequest.Stream {
		return c.String(http.StatusOK, "")
	}

	var syncedPeers = make(map[uint64]bool)

	response, syncedPeers, err := h.createMapResponse(m, binder, mapRequest, false, make(map[uint64]bool))
	if err != nil {
		return err
	}

	updateChan := make(chan *broker.Signal, 20)
	client := broker.NewClient(machineID, updateChan)

	tailnetBroker.AddClient(&client)
	h.cancelOfflineMessage(machineID)

	// Listen to connection close and un-register messageChan
	notify := c.Request().Context().Done()

	keepAliveResponse, err := h.createKeepAliveResponse(binder, mapRequest)
	if err != nil {
		return err
	}
	keepAliveTicker := time.NewTicker(keepAliveInterval)
	syncTicker := time.NewTicker(5 * time.Second)

	var latestSync = time.Now()
	var latestUpdate = latestSync

	c.Response().WriteHeader(http.StatusOK)

	if _, err := c.Response().Write(response); err != nil {
		return err
	}
	c.Response().Flush()

	defer func() {
		tailnetBroker.RemoveClient(machineID)
		keepAliveTicker.Stop()
		syncTicker.Stop()
		_ = h.repository.SetMachineLastSeen(ctx, machineID)
		h.scheduleOfflineMessage(tailnetID, machineID)
	}()

	for {
		select {
		case s := <-updateChan:
			if s.PeerUpdated == nil || *s.PeerUpdated != machineID {
				latestUpdate = time.Now()
			}
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

				payload, syncedPeers, payloadErr = h.createMapResponse(machine, binder, mapRequest, true, syncedPeers)

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

	response, _, err := h.createMapResponse(m, binder, request, false, map[uint64]bool{})
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

func (h *PollNetMapHandler) createMapResponse(m *domain.Machine, binder bind.Binder, request *tailcfg.MapRequest, delta bool, prevSyncedPeerIDs map[uint64]bool) ([]byte, map[uint64]bool, error) {
	ctx := context.TODO()

	node, err := mapping.ToNode(m, true)
	if err != nil {
		return nil, nil, err
	}

	policies, err := h.repository.GetACLPolicy(ctx, m.TailnetID)
	if err != nil {
		return nil, nil, err
	}

	users, err := h.repository.ListUsers(ctx, m.TailnetID)
	if err != nil {
		return nil, nil, err
	}

	var changedPeers []*tailcfg.Node
	var removedPeers []tailcfg.NodeID

	candidatePeers, err := h.repository.ListMachinePeers(ctx, m.TailnetID, m.MachineKey)
	if err != nil {
		return nil, nil, err
	}

	syncedPeerIDs := map[uint64]bool{}

	for _, peer := range candidatePeers {
		if domain.IsValidPeer(policies, m, &peer) || domain.IsValidPeer(policies, &peer, m) {
			n, err := mapping.ToNode(&peer, h.brokers(peer.TailnetID).IsConnected(peer.ID))
			if err != nil {
				return nil, nil, err
			}
			changedPeers = append(changedPeers, n)
			syncedPeerIDs[peer.ID] = true
			delete(prevSyncedPeerIDs, peer.ID)
		}
	}

	for p, _ := range prevSyncedPeerIDs {
		removedPeers = append(removedPeers, tailcfg.NodeID(p))
	}

	dnsConfig, err := h.repository.GetDNSConfig(ctx, m.TailnetID)
	if err != nil {
		return nil, nil, err
	}

	derpMap, err := h.repository.GetDERPMap(ctx)
	if err != nil {
		return nil, nil, err
	}

	rules := domain.BuildFilterRules(policies, m, candidatePeers)

	controlTime := time.Now().UTC()
	var mapResponse *tailcfg.MapResponse

	if !delta {
		mapResponse = &tailcfg.MapResponse{
			KeepAlive:    false,
			Node:         node,
			DNSConfig:    mapping.ToDNSConfig(&m.Tailnet, dnsConfig),
			PacketFilter: rules,
			DERPMap:      derpMap,
			Domain:       dnsname.SanitizeHostname(m.Tailnet.Name),
			Peers:        changedPeers,
			UserProfiles: mapping.ToUserProfiles(users),
			ControlTime:  &controlTime,
		}
	} else {
		mapResponse = &tailcfg.MapResponse{
			PacketFilter: rules,
			PeersChanged: changedPeers,
			PeersRemoved: removedPeers,
			UserProfiles: mapping.ToUserProfiles(users),
			ControlTime:  &controlTime,
		}
	}

	if request.OmitPeers {
		mapResponse.PeersChanged = nil
		mapResponse.PeersRemoved = nil
		mapResponse.Peers = nil
	}

	payload, err := binder.Marshal(request.Compress, mapResponse)

	return payload, syncedPeerIDs, nil
}

func NewOfflineTimers(repository domain.Repository, brokers *broker.BrokerPool) *OfflineTimers {
	return &OfflineTimers{
		repository: repository,
		brokers:    brokers.Get,
		data:       make(map[uint64]*time.Timer),
		startCh:    make(chan [2]uint64),
		stopCh:     make(chan uint64),
	}
}

type OfflineTimers struct {
	repository domain.Repository
	brokers    func(uint64) broker.Broker
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

	timer := time.NewTimer(10 * time.Second)
	go func() {
		<-timer.C
		if !o.brokers(tailnetID).IsConnected(machineID) {
			o.brokers(tailnetID).SignalPeerUpdated(machineID)
			o.stopCh <- machineID
		}
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
