package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/netip"
	"time"
)

func (s *Service) machineToApi(m *domain.Machine) *api.Machine {
	var lastSeen *timestamppb.Timestamp

	var name = m.Name
	if m.NameIdx != 0 {
		name = fmt.Sprintf("%s-%d", m.Name, m.NameIdx)
	}

	online := false
	if m.LastSeen != nil {
		lastSeen = timestamppb.New(*m.LastSeen)
		online = m.LastSeen.After(time.Now().Add(-config.KeepAliveInterval()))
	}

	return &api.Machine{
		Id:                m.ID,
		Name:              name,
		Ipv4:              m.IPv4.String(),
		Ipv6:              m.IPv6.String(),
		Ephemeral:         m.Ephemeral,
		Tags:              m.Tags,
		LastSeen:          lastSeen,
		CreatedAt:         timestamppb.New(m.CreatedAt),
		ExpiresAt:         timestamppb.New(m.ExpiresAt),
		KeyExpiryDisabled: m.KeyExpiryDisabled,
		Connected:         online,
		Os:                m.HostInfo.OS,
		ClientVersion:     m.HostInfo.IPNVersion,
		Tailnet: &api.Ref{
			Id:   m.Tailnet.ID,
			Name: m.Tailnet.Name,
		},
		User: &api.Ref{
			Id:   m.User.ID,
			Name: m.User.Name,
		},
		ClientConnectivity: &api.ClientConnectivity{
			Endpoints: m.Endpoints,
		},
		AdvertisedRoutes:   m.AdvertisedPrefixes(),
		EnabledRoutes:      m.AllowedPrefixes(),
		AdvertisedExitNode: m.IsAdvertisedExitNode(),
		EnabledExitNode:    m.IsAllowedExitNode(),
	}
}

func (s *Service) ListMachines(ctx context.Context, req *connect.Request[api.ListMachinesRequest]) (*connect.Response[api.ListMachinesResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	machines, err := s.repository.ListMachineByTailnet(ctx, tailnet.ID)
	if err != nil {
		return nil, err
	}

	response := &api.ListMachinesResponse{}
	for _, m := range machines {
		response.Machines = append(response.Machines, s.machineToApi(&m))
	}

	return connect.NewResponse(response), nil
}

func (s *Service) GetMachine(ctx context.Context, req *connect.Request[api.GetMachineRequest]) (*connect.Response[api.GetMachineResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	return connect.NewResponse(&api.GetMachineResponse{Machine: s.machineToApi(m)}), nil
}

func (s *Service) DeleteMachine(ctx context.Context, req *connect.Request[api.DeleteMachineRequest]) (*connect.Response[api.DeleteMachineResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if _, err := s.repository.DeleteMachine(ctx, req.Msg.MachineId); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeersRemoved: []uint64{m.ID}})

	return connect.NewResponse(&api.DeleteMachineResponse{}), nil
}

func (s *Service) ExpireMachine(ctx context.Context, req *connect.Request[api.ExpireMachineRequest]) (*connect.Response[api.ExpireMachineResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	timestamp := time.Unix(123, 0)
	m.ExpiresAt = timestamp
	m.KeyExpiryDisabled = false

	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	return connect.NewResponse(&api.ExpireMachineResponse{}), nil
}

func (s *Service) GetMachineRoutes(ctx context.Context, req *connect.Request[api.GetMachineRoutesRequest]) (*connect.Response[api.GetMachineRoutesResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	response := api.GetMachineRoutesResponse{
		MachineId: m.ID,
		Routes: &api.MachineRoutes{
			AdvertisedRoutes:   m.AdvertisedPrefixes(),
			EnabledRoutes:      m.AllowedPrefixes(),
			AdvertisedExitNode: m.IsAdvertisedExitNode(),
			EnabledExitNode:    m.IsAllowedExitNode(),
		},
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) EnableMachineRoutes(ctx context.Context, req *connect.Request[api.EnableMachineRoutesRequest]) (*connect.Response[api.EnableMachineRoutesResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	var allowIPs = domain.NewAllowIPsSet(m.AllowIPs)
	var autoAllowIPs = domain.NewAllowIPsSet(m.AutoAllowIPs)

	if req.Msg.Replace {
		allowIPs = domain.NewAllowIPsSet([]netip.Prefix{})
		autoAllowIPs = domain.NewAllowIPsSet([]netip.Prefix{})
	}

	for _, r := range req.Msg.Routes {
		prefix, err := netip.ParsePrefix(r)
		if err != nil {
			return nil, err
		}
		allowIPs.Add(prefix)
	}

	m.AllowIPs = allowIPs.Items()
	m.AutoAllowIPs = autoAllowIPs.Items()
	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	response := api.EnableMachineRoutesResponse{
		MachineId: m.ID,
		Routes: &api.MachineRoutes{
			AdvertisedRoutes:   m.AdvertisedPrefixes(),
			EnabledRoutes:      m.AllowedPrefixes(),
			AdvertisedExitNode: m.IsAdvertisedExitNode(),
			EnabledExitNode:    m.IsAllowedExitNode(),
		},
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) DisableMachineRoutes(ctx context.Context, req *connect.Request[api.DisableMachineRoutesRequest]) (*connect.Response[api.DisableMachineRoutesResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	allowIPs := domain.NewAllowIPsSet(m.AllowIPs)
	autoAllowIPs := domain.NewAllowIPsSet(m.AutoAllowIPs)

	for _, r := range req.Msg.Routes {
		prefix, err := netip.ParsePrefix(r)
		if err != nil {
			return nil, err
		}
		allowIPs.Remove(prefix)
		autoAllowIPs.Remove(prefix)
	}

	m.AllowIPs = allowIPs.Items()
	m.AutoAllowIPs = autoAllowIPs.Items()
	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	response := api.DisableMachineRoutesResponse{
		MachineId: m.ID,
		Routes: &api.MachineRoutes{
			AdvertisedRoutes:   m.AdvertisedPrefixes(),
			EnabledRoutes:      m.AllowedPrefixes(),
			AdvertisedExitNode: m.IsAdvertisedExitNode(),
			EnabledExitNode:    m.IsAllowedExitNode(),
		},
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) EnableExitNode(ctx context.Context, req *connect.Request[api.EnableExitNodeRequest]) (*connect.Response[api.EnableExitNodeResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if !m.IsAdvertisedExitNode() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("machine is not a valid exit node"))
	}

	prefix4 := netip.MustParsePrefix("0.0.0.0/0")
	prefix6 := netip.MustParsePrefix("::/0")

	allowIPs := domain.NewAllowIPsSet(m.AllowIPs)
	allowIPs.Add(prefix4, prefix6)

	m.AllowIPs = allowIPs.Items()

	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	response := api.EnableExitNodeResponse{
		MachineId: m.ID,
		Routes: &api.MachineRoutes{
			AdvertisedRoutes:   m.AdvertisedPrefixes(),
			EnabledRoutes:      m.AllowedPrefixes(),
			AdvertisedExitNode: m.IsAdvertisedExitNode(),
			EnabledExitNode:    m.IsAllowedExitNode(),
		},
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) DisableExitNode(ctx context.Context, req *connect.Request[api.DisableExitNodeRequest]) (*connect.Response[api.DisableExitNodeResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if !m.IsAdvertisedExitNode() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("machine is not a valid exit node"))
	}

	prefix4 := netip.MustParsePrefix("0.0.0.0/0")
	prefix6 := netip.MustParsePrefix("::/0")

	allowIPs := domain.NewAllowIPsSet(m.AllowIPs)
	allowIPs.Remove(prefix4, prefix6)

	autoAllowIPs := domain.NewAllowIPsSet(m.AutoAllowIPs)
	autoAllowIPs.Remove(prefix4, prefix6)

	m.AllowIPs = allowIPs.Items()
	m.AutoAllowIPs = autoAllowIPs.Items()

	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	response := api.DisableExitNodeResponse{
		MachineId: m.ID,
		Routes: &api.MachineRoutes{
			AdvertisedRoutes:   m.AdvertisedPrefixes(),
			EnabledRoutes:      m.AllowedPrefixes(),
			AdvertisedExitNode: m.IsAdvertisedExitNode(),
			EnabledExitNode:    m.IsAllowedExitNode(),
		},
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) SetMachineKeyExpiry(ctx context.Context, req *connect.Request[api.SetMachineKeyExpiryRequest]) (*connect.Response[api.SetMachineKeyExpiryResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	m.KeyExpiryDisabled = req.Msg.Disabled

	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.pubsub.Publish(m.TailnetID, &broker.Signal{PeerUpdated: &m.ID})

	return connect.NewResponse(&api.SetMachineKeyExpiryResponse{}), nil
}
