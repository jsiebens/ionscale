package service

import (
	"context"
	"encoding/json"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) GetDERPMap(ctx context.Context, req *connect.Request[api.GetDERPMapRequest]) (*connect.Response[api.GetDERPMapResponse], error) {
	derpMap, err := s.repository.GetDERPMap(ctx)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(derpMap)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetDERPMapResponse{Value: raw}), nil
}

func (s *Service) SetDERPMap(ctx context.Context, req *connect.Request[api.SetDERPMapRequest]) (*connect.Response[api.SetDERPMapResponse], error) {
	var derpMap tailcfg.DERPMap
	err := json.Unmarshal(req.Msg.Value, &derpMap)
	if err != nil {
		return nil, err
	}

	if err := s.repository.SetDERPMap(ctx, &derpMap); err != nil {
		return nil, err
	}

	s.brokerPool.SignalDERPMapUpdated(&derpMap)

	return connect.NewResponse(&api.SetDERPMapResponse{Value: req.Msg.Value}), nil
}
