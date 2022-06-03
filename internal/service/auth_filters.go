package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/hashicorp/go-bexpr"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) ListAuthFilters(ctx context.Context, req *connect.Request[api.ListAuthFiltersRequest]) (*connect.Response[api.ListAuthFiltersResponse], error) {
	response := &api.ListAuthFiltersResponse{AuthFilters: []*api.AuthFilter{}}

	if req.Msg.AuthMethodId == nil {
		filters, err := s.repository.ListAuthFilters(ctx)
		if err != nil {
			return nil, err
		}
		for _, filter := range filters {
			response.AuthFilters = append(response.AuthFilters, s.mapToApi(&filter.AuthMethod, filter))
		}
	} else {
		authMethod, err := s.repository.GetAuthMethod(ctx, *req.Msg.AuthMethodId)
		if err != nil {
			return nil, err
		}
		if authMethod == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invalid auth method id"))
		}

		filters, err := s.repository.ListAuthFiltersByAuthMethod(ctx, authMethod.ID)
		if err != nil {
			return nil, err
		}
		for _, filter := range filters {
			response.AuthFilters = append(response.AuthFilters, s.mapToApi(&filter.AuthMethod, filter))
		}
	}

	return connect.NewResponse[api.ListAuthFiltersResponse](response), nil
}

func (s *Service) CreateAuthFilter(ctx context.Context, req *connect.Request[api.CreateAuthFilterRequest]) (*connect.Response[api.CreateAuthFilterResponse], error) {
	authMethod, err := s.repository.GetAuthMethod(ctx, req.Msg.AuthMethodId)
	if err != nil {
		return nil, err
	}
	if authMethod == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invalid auth method id"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invalid tailnet id"))
	}

	if req.Msg.Expr != "*" {
		if _, err := bexpr.CreateEvaluator(req.Msg.Expr); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid expression: %v", err))
		}
	}

	authFilter := &domain.AuthFilter{
		ID:         util.NextID(),
		Expr:       req.Msg.Expr,
		AuthMethod: *authMethod,
		Tailnet:    tailnet,
	}

	if err := s.repository.SaveAuthFilter(ctx, authFilter); err != nil {
		return nil, err
	}

	response := api.CreateAuthFilterResponse{AuthFilter: s.mapToApi(authMethod, *authFilter)}

	return connect.NewResponse[api.CreateAuthFilterResponse](&response), nil
}

func (s *Service) DeleteAuthFilter(ctx context.Context, req *connect.Request[api.DeleteAuthFilterRequest]) (*connect.Response[api.DeleteAuthFilterResponse], error) {

	err := s.repository.Transaction(func(rp domain.Repository) error {

		filter, err := rp.GetAuthFilter(ctx, req.Msg.AuthFilterId)
		if err != nil {
			return err
		}

		if filter == nil {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("auth filter not found"))
		}

		c, err := rp.ExpireMachineByAuthMethod(ctx, *filter.TailnetID, filter.AuthMethodID)
		if err != nil {
			return err
		}

		if err := rp.DeleteAuthFilter(ctx, filter.ID); err != nil {
			return err
		}

		if c != 0 {
			s.brokers(*filter.TailnetID).SignalUpdate()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	response := api.DeleteAuthFilterResponse{}

	return connect.NewResponse[api.DeleteAuthFilterResponse](&response), nil
}

func (s *Service) mapToApi(authMethod *domain.AuthMethod, filter domain.AuthFilter) *api.AuthFilter {
	result := api.AuthFilter{
		Id:   filter.ID,
		Expr: filter.Expr,
		AuthMethod: &api.Ref{
			Id:   authMethod.ID,
			Name: authMethod.Name,
		},
	}

	if filter.Tailnet != nil {
		id := filter.Tailnet.ID
		name := filter.Tailnet.Name

		result.Tailnet = &api.Ref{
			Id:   id,
			Name: name,
		}
	}

	return &result
}
