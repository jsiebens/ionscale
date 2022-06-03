package service

import (
	"context"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) CreateAuthMethod(ctx context.Context, req *connect.Request[api.CreateAuthMethodRequest]) (*connect.Response[api.CreateAuthMethodResponse], error) {

	authMethod := &domain.AuthMethod{
		ID:           util.NextID(),
		Name:         req.Msg.Name,
		Type:         req.Msg.Type,
		Issuer:       req.Msg.Issuer,
		ClientId:     req.Msg.ClientId,
		ClientSecret: req.Msg.ClientSecret,
	}

	if err := s.repository.SaveAuthMethod(ctx, authMethod); err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.CreateAuthMethodResponse{AuthMethod: &api.AuthMethod{
		Id:       authMethod.ID,
		Type:     authMethod.Type,
		Name:     authMethod.Name,
		Issuer:   authMethod.Issuer,
		ClientId: authMethod.ClientId,
	}}), nil

}

func (s *Service) ListAuthMethods(ctx context.Context, _ *connect.Request[api.ListAuthMethodsRequest]) (*connect.Response[api.ListAuthMethodsResponse], error) {
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

	return connect.NewResponse(response), nil
}
