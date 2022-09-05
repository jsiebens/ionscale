package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) GetDERPMap(ctx context.Context, _ *connect.Request[api.GetDERPMapRequest]) (*connect.Response[api.GetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

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
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	var derpMap tailcfg.DERPMap
	err := json.Unmarshal(req.Msg.Value, &derpMap)
	if err != nil {
		return nil, err
	}

	tailnets, err := s.repository.ListTailnets(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.repository.SetDERPMap(ctx, &derpMap); err != nil {
		return nil, err
	}

	for _, t := range tailnets {
		s.pubsub.Publish(t.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.SetDERPMapResponse{Value: req.Msg.Value}), nil
}
