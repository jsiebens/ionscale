package service

import (
	"context"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/jsiebens/ionscale/pkg/gen/api"
)

func (s *Service) CreateAuthMethod(ctx context.Context, req *api.CreateAuthMethodRequest) (*api.CreateAuthMethodResponse, error) {

	authMethod := &domain.AuthMethod{
		ID:           util.NextID(),
		Name:         req.Name,
		Type:         req.Type,
		Issuer:       req.Issuer,
		ClientId:     req.ClientId,
		ClientSecret: req.ClientSecret,
	}

	if err := s.repository.SaveAuthMethod(ctx, authMethod); err != nil {
		return nil, err
	}

	return &api.CreateAuthMethodResponse{AuthMethod: &api.AuthMethod{
		Id:       authMethod.ID,
		Type:     authMethod.Type,
		Name:     authMethod.Name,
		Issuer:   authMethod.Issuer,
		ClientId: authMethod.ClientId,
	}}, nil

}

func (s *Service) ListAuthMethods(ctx context.Context, _ *api.ListAuthMethodsRequest) (*api.ListAuthMethodsResponse, error) {
	methods, err := s.repository.ListAuthMethods(ctx)
	if err != nil {
		return nil, err
	}

	response := &api.ListAuthMethodsResponse{AuthMethods: []*api.AuthMethod{}}
	for _, m := range methods {
		response.AuthMethods = append(response.AuthMethods, &api.AuthMethod{
			Id:   m.ID,
			Name: m.Name,
			Type: m.Type,
		})
	}

	return response, nil
}
