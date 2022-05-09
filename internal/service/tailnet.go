package service

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
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
