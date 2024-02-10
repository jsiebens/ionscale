package mapping

import (
	"context"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/domain"
	"net/netip"
	"sync"
	"tailscale.com/tailcfg"
	"tailscale.com/types/opt"
	"time"
)

// MapResponse is a custom tailcfg.MapResponse
// for marshalling non-nil zero-length slices (meaning explicitly now empty)
// see tailcfg.MapResponse documentation
type MapResponse struct {
	tailcfg.MapResponse
	PacketFilter []tailcfg.FilterRule
}

func NewPollNetMapper(req *tailcfg.MapRequest, machineID uint64, repository domain.Repository, sessionManager core.PollMapSessionManager) *PollNetMapper {
	return &PollNetMapper{
		req:                 req,
		machineID:           machineID,
		prevSyncedPeerIDs:   make(map[uint64]bool),
		prevDerpMapChecksum: "",
		repository:          repository,
		sessionManager:      sessionManager,
	}
}

type PollNetMapper struct {
	sync.Mutex
	req       *tailcfg.MapRequest
	machineID uint64

	prevSyncedPeerIDs   map[uint64]bool
	prevDerpMapChecksum string

	repository     domain.Repository
	sessionManager core.PollMapSessionManager
}

func (h *PollNetMapper) CreateMapResponse(ctx context.Context, delta bool) (*MapResponse, error) {
	h.Lock()
	defer h.Unlock()

	m, err := h.repository.GetMachine(ctx, h.machineID)
	if err != nil {
		return nil, err
	}

	hostinfo := tailcfg.Hostinfo(m.HostInfo)
	tailnet := m.Tailnet
	policies := tailnet.ACLPolicy
	dnsConfig := tailnet.DNSConfig

	serviceUser, _, err := h.repository.GetOrCreateServiceUser(ctx, &tailnet)
	if err != nil {
		return nil, err
	}

	derpMap, err := m.Tailnet.GetDERPMap(ctx, h.repository)
	if err != nil {
		return nil, err
	}

	prc := &primaryRoutesCollector{flagged: map[netip.Prefix]bool{}}

	node, user, err := ToNode(h.req.Version, m, &tailnet, serviceUser, false, true, prc.filter)
	if err != nil {
		return nil, err
	}

	var users = []tailcfg.UserProfile{*user}
	var changedPeers []*tailcfg.Node
	var removedPeers []tailcfg.NodeID
	var filterRules = make([]tailcfg.FilterRule, 0)
	var sshPolicy *tailcfg.SSHPolicy
	syncedPeerIDs := map[uint64]bool{}

	if !h.req.OmitPeers {
		candidatePeers, err := h.repository.ListMachinePeers(ctx, m.TailnetID, m.ID)
		if err != nil {
			return nil, err
		}

		syncedUserIDs := map[tailcfg.UserID]bool{user.ID: true}

		for _, peer := range candidatePeers {
			if peer.IsExpired() {
				continue
			}
			if policies.IsValidPeer(m, &peer) || policies.IsValidPeer(&peer, m) {
				isConnected := h.sessionManager.HasSession(peer.TailnetID, peer.ID)

				n, u, err := ToNode(h.req.Version, &peer, &tailnet, serviceUser, true, isConnected, prc.filter)
				if err != nil {
					return nil, err
				}
				changedPeers = append(changedPeers, n)
				syncedPeerIDs[peer.ID] = true
				delete(h.prevSyncedPeerIDs, peer.ID)

				if _, ok := syncedUserIDs[u.ID]; !ok {
					users = append(users, *u)
					syncedUserIDs[u.ID] = true
				}
			}
		}

		for p, _ := range h.prevSyncedPeerIDs {
			removedPeers = append(removedPeers, tailcfg.NodeID(p))
		}

		filterRules = policies.BuildFilterRules(candidatePeers, m)

		if tailnet.SSHEnabled && hostinfo.TailscaleSSHEnabled() {
			sshPolicy = policies.BuildSSHPolicy(candidatePeers, m)
		}
	}

	controlTime := time.Now().UTC()
	var mapResponse tailcfg.MapResponse

	if !delta {
		mapResponse = tailcfg.MapResponse{
			KeepAlive:       false,
			Node:            node,
			DNSConfig:       ToDNSConfig(m, &m.Tailnet, &dnsConfig),
			PacketFilter:    filterRules,
			SSHPolicy:       sshPolicy,
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
		mapResponse = tailcfg.MapResponse{
			Node:            node,
			DNSConfig:       ToDNSConfig(m, &m.Tailnet, &dnsConfig),
			PacketFilter:    filterRules,
			SSHPolicy:       sshPolicy,
			Domain:          domain.SanitizeTailnetName(m.Tailnet.Name),
			PeersChanged:    changedPeers,
			PeersRemoved:    removedPeers,
			UserProfiles:    users,
			ControlTime:     &controlTime,
			CollectServices: optBool(tailnet.ServiceCollectionEnabled),
		}

		if h.prevDerpMapChecksum != derpMap.Checksum {
			mapResponse.DERPMap = &derpMap.DERPMap
		}
	}

	if h.req.OmitPeers {
		mapResponse.PeersChanged = nil
		mapResponse.PeersRemoved = nil
		mapResponse.Peers = nil
	}

	h.prevSyncedPeerIDs = syncedPeerIDs
	h.prevDerpMapChecksum = derpMap.Checksum

	return &MapResponse{MapResponse: mapResponse, PacketFilter: filterRules}, nil
}

type primaryRoutesCollector struct {
	flagged map[netip.Prefix]bool
}

func (p *primaryRoutesCollector) filter(m *domain.Machine) []netip.Prefix {
	var result []netip.Prefix
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

func optBool(v bool) opt.Bool {
	b := opt.Bool("")
	b.Set(v)
	return b
}
