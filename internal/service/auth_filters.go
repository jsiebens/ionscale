package service

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-bexpr"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) ListAuthFilters(ctx context.Context, req *api.ListAuthFiltersRequest) (*api.ListAuthFiltersResponse, error) {
	response := &api.ListAuthFiltersResponse{AuthFilters: []*api.AuthFilter{}}

	if req.AuthMethodId == nil {
		filters, err := s.repository.ListAuthFilters(ctx)
		if err != nil {
			return nil, err
		}
		for _, filter := range filters {
			response.AuthFilters = append(response.AuthFilters, s.mapToApi(&filter.AuthMethod, filter))
		}
	} else {
		authMethod, err := s.repository.GetAuthMethod(ctx, *req.AuthMethodId)
		if err != nil {
			return nil, err
		}
		if authMethod == nil {
			return nil, status.Error(codes.NotFound, "invalid auth method id")
		}

		filters, err := s.repository.ListAuthFiltersByAuthMethod(ctx, authMethod.ID)
		if err != nil {
			return nil, err
		}
		for _, filter := range filters {
			response.AuthFilters = append(response.AuthFilters, s.mapToApi(&filter.AuthMethod, filter))
		}
	}

	return response, nil
}

func (s *Service) CreateAuthFilter(ctx context.Context, req *api.CreateAuthFilterRequest) (*api.CreateAuthFilterResponse, error) {
	authMethod, err := s.repository.GetAuthMethod(ctx, req.AuthMethodId)
	if err != nil {
		return nil, err
	}
	if authMethod == nil {
		return nil, status.Error(codes.NotFound, "invalid auth method id")
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "invalid tailnet id")
	}

	if req.Expr != "*" {
		if _, err := bexpr.CreateEvaluator(req.Expr); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid expression: %v", err))
		}
	}

	authFilter := &domain.AuthFilter{
		ID:         util.NextID(),
		Expr:       req.Expr,
		AuthMethod: *authMethod,
		Tailnet:    tailnet,
	}

	if err := s.repository.SaveAuthFilter(ctx, authFilter); err != nil {
		return nil, err
	}

	response := api.CreateAuthFilterResponse{AuthFilter: s.mapToApi(authMethod, *authFilter)}

	return &response, nil
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
