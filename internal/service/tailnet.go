package service

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CreateTailnet(ctx context.Context, req *api.CreateTailnetRequest) (*api.CreateTailnetResponse, error) {
	tailnet, created, err := s.repository.GetOrCreateTailnet(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	if !created {
		return nil, fmt.Errorf("tailnet already exists")
	}

	resp := &api.CreateTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}

	return resp, nil
}

func (s *Service) GetTailnet(ctx context.Context, req *api.GetTailnetRequest) (*api.GetTailnetResponse, error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "")
	}

	return &api.GetTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}, nil
}

func (s *Service) ListTailnets(ctx context.Context, _ *api.ListTailnetRequest) (*api.ListTailnetResponse, error) {
	resp := &api.ListTailnetResponse{}

	tailnets, err := s.repository.ListTailnets(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tailnets {
		gt := api.Tailnet{Id: t.ID, Name: t.Name}
		resp.Tailnet = append(resp.Tailnet, &gt)
	}
	return resp, nil
}

func (s *Service) DeleteTailnet(ctx context.Context, req *api.DeleteTailnetRequest) (*api.DeleteTailnetResponse, error) {

	count, err := s.repository.CountMachineByTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	if !req.Force && count > 0 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("tailnet is not empty, number of machines: %d", count))
	}

	err = s.repository.Transaction(func(tx domain.Repository) error {
		if err := tx.DeleteMachineByTailnet(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteAuthKeysByTailnet(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteUsersByTailnet(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteAuthFiltersByTailnet(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteACLPolicy(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteDNSConfig(ctx, req.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteTailnet(ctx, req.TailnetId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.brokers(req.TailnetId).SignalUpdate()

	return &api.DeleteTailnetResponse{}, nil
}
