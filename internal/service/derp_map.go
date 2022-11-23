package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/errors"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) GetDefaultDERPMap(ctx context.Context, _ *connect.Request[api.GetDefaultDERPMapRequest]) (*connect.Response[api.GetDefaultDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	dm, err := s.repository.GetDERPMap(ctx)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	raw, err := json.Marshal(dm.DERPMap)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return connect.NewResponse(&api.GetDefaultDERPMapResponse{Value: raw}), nil
}

func (s *Service) SetDefaultDERPMap(ctx context.Context, req *connect.Request[api.SetDefaultDERPMapRequest]) (*connect.Response[api.SetDefaultDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	var derpMap tailcfg.DERPMap
	if err := json.Unmarshal(req.Msg.Value, &derpMap); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	dp := domain.DERPMap{
		Checksum: util.Checksum(&derpMap),
		DERPMap:  derpMap,
	}

	if err := s.repository.SetDERPMap(ctx, &dp); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	tailnets, err := s.repository.ListTailnets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, t := range tailnets {
		s.pubsub.Publish(t.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.SetDefaultDERPMapResponse{Value: req.Msg.Value}), nil
}

func (s *Service) ResetDefaultDERPMap(ctx context.Context, req *connect.Request[api.ResetDefaultDERPMapRequest]) (*connect.Response[api.ResetDefaultDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	dp := domain.DERPMap{}

	if err := s.repository.SetDERPMap(ctx, &dp); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	tailnets, err := s.repository.ListTailnets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, t := range tailnets {
		s.pubsub.Publish(t.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.ResetDefaultDERPMapResponse{}), nil
}
