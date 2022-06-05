package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"inet.af/netaddr"
	"time"
)

func (s *Service) ListMachines(ctx context.Context, req *connect.Request[api.ListMachinesRequest]) (*connect.Response[api.ListMachinesResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.TailnetMatches(req.Msg.TailnetId) {
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
		var name = m.Name
		if m.NameIdx != 0 {
			name = fmt.Sprintf("%s-%d", m.Name, m.NameIdx)
		}
		online := s.brokers(m.TailnetID).IsConnected(m.ID)
		var lastSeen *timestamppb.Timestamp
		if m.LastSeen != nil {
			lastSeen = timestamppb.New(*m.LastSeen)
		}
		response.Machines = append(response.Machines, &api.Machine{
			Id:        m.ID,
			Name:      name,
			Ipv4:      m.IPv4.String(),
			Ipv6:      m.IPv6.String(),
			Ephemeral: m.Ephemeral,
			Tags:      m.Tags,
			LastSeen:  lastSeen,
			Connected: online,
			Tailnet: &api.Ref{
				Id:   m.Tailnet.ID,
				Name: m.Tailnet.Name,
			},
			User: &api.Ref{
				Id:   m.User.ID,
				Name: m.User.Name,
			},
		})
	}

	return connect.NewResponse(response), nil
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

	if !principal.IsSystemAdmin() && !principal.UserMatches(m.UserID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if _, err := s.repository.DeleteMachine(ctx, req.Msg.MachineId); err != nil {
		return nil, err
	}

	s.brokers(m.TailnetID).SignalPeersRemoved([]uint64{m.ID})

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

	if !principal.IsSystemAdmin() && !principal.UserMatches(m.UserID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	timestamp := time.Unix(123, 0)
	m.ExpiresAt = &timestamp

	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.brokers(m.TailnetID).SignalPeerUpdated(m.ID)

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

	if !principal.IsSystemAdmin() && !principal.TailnetMatches(m.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	var routes []*api.RoutableIP
	for _, r := range m.HostInfo.RoutableIPs {
		routes = append(routes, &api.RoutableIP{
			Advertised: r.String(),
			Allowed:    m.IsAllowedIPPrefix(r),
		})
	}

	response := api.GetMachineRoutesResponse{
		Routes: routes,
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) SetMachineRoutes(ctx context.Context, req *connect.Request[api.SetMachineRoutesRequest]) (*connect.Response[api.GetMachineRoutesResponse], error) {
	principal := CurrentPrincipal(ctx)

	m, err := s.repository.GetMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("machine not found"))
	}

	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	var allowedIps []netaddr.IPPrefix
	for _, r := range req.Msg.AllowedIps {
		prefix, err := netaddr.ParseIPPrefix(r)
		if err != nil {
			return nil, err
		}
		allowedIps = append(allowedIps, prefix)
	}

	m.AllowIPs = allowedIps
	if err := s.repository.SaveMachine(ctx, m); err != nil {
		return nil, err
	}

	s.brokers(m.TailnetID).SignalPeerUpdated(m.ID)

	var routes []*api.RoutableIP
	for _, r := range m.HostInfo.RoutableIPs {
		routes = append(routes, &api.RoutableIP{
			Advertised: r.String(),
			Allowed:    m.IsAllowedIPPrefix(r),
		})
	}

	response := api.GetMachineRoutesResponse{
		Routes: routes,
	}

	return connect.NewResponse(&response), nil
}
