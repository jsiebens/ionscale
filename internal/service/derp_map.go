package service

import (
	"context"
	"encoding/json"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"tailscale.com/tailcfg"
)

func (s *Service) GetDERPMap(ctx context.Context, req *api.GetDERPMapRequest) (*api.GetDERPMapResponse, error) {
	derpMap, err := s.repository.GetDERPMap(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(derpMap)
	if err != nil {
		return nil, err
	}

	return &api.GetDERPMapResponse{Value: raw}, nil
}

func (s *Service) SetDERPMap(ctx context.Context, req *api.SetDERPMapRequest) (*api.SetDERPMapResponse, error) {
	var derpMap tailcfg.DERPMap
	err := json.Unmarshal(req.Value, &derpMap)
	if err != nil {
		return nil, err
	}

	if err := s.repository.SetDERPMap(ctx, &derpMap); err != nil {
		return nil, err
	}

	s.brokerPool.SignalDERPMapUpdated(&derpMap)

	return &api.SetDERPMapResponse{Value: req.Value}, nil
}
