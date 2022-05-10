package service

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) ListMachines(ctx context.Context, req *api.ListMachinesRequest) (*api.ListMachinesResponse, error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "tailnet does not exist")
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
			Ipv4:      m.IPv4,
			Ipv6:      m.IPv6,
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

	return response, nil
}

func (s *Service) DeleteMachine(ctx context.Context, req *api.DeleteMachineRequest) (*api.DeleteMachineResponse, error) {
	m, err := s.repository.GetMachine(ctx, req.MachineId)
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, status.Error(codes.NotFound, "machine does not exist")
	}

	if _, err := s.repository.DeleteMachine(ctx, req.MachineId); err != nil {
		return nil, err
	}

	s.brokers(m.TailnetID).SignalPeersRemoved([]uint64{m.ID})

	return &api.DeleteMachineResponse{}, nil
}
