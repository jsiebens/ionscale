package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) CreateTailnet(ctx context.Context, req *connect.Request[api.CreateTailnetRequest]) (*connect.Response[api.CreateTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, created, err := s.repository.GetOrCreateTailnet(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	if !created {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tailnet already exists"))
	}

	resp := &api.CreateTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}

	return connect.NewResponse(resp), nil
}

func (s *Service) GetTailnet(ctx context.Context, req *connect.Request[api.GetTailnetRequest]) (*connect.Response[api.GetTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.TailnetMatches(req.Msg.Id) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	return connect.NewResponse(&api.GetTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}), nil
}

func (s *Service) ListTailnets(ctx context.Context, req *connect.Request[api.ListTailnetRequest]) (*connect.Response[api.ListTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)

	resp := &api.ListTailnetResponse{}

	if principal.IsSystemAdmin() {
		tailnets, err := s.repository.ListTailnets(ctx)
		if err != nil {
			return nil, err
		}
		for _, t := range tailnets {
			gt := api.Tailnet{Id: t.ID, Name: t.Name}
			resp.Tailnet = append(resp.Tailnet, &gt)
		}
	}

	if principal.User != nil {
		tailnet, err := s.repository.GetTailnet(ctx, principal.User.TailnetID)
		if err != nil {
			return nil, err
		}
		gt := api.Tailnet{Id: tailnet.ID, Name: tailnet.Name}
		resp.Tailnet = append(resp.Tailnet, &gt)
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) DeleteTailnet(ctx context.Context, req *connect.Request[api.DeleteTailnetRequest]) (*connect.Response[api.DeleteTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.TailnetMatches(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	count, err := s.repository.CountMachineByTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if !req.Msg.Force && count > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tailnet is not empty, number of machines: %d", count))
	}

	err = s.repository.Transaction(func(tx domain.Repository) error {
		if err := tx.DeleteMachineByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteAuthKeysByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteUsersByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteACLPolicy(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteDNSConfig(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.brokers(req.Msg.TailnetId).SignalUpdate()

	return connect.NewResponse(&api.DeleteTailnetResponse{}), nil
}
