package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetAuthMethod(ctx context.Context, req *connect.Request[api.GetAuthMethodRequest]) (*connect.Response[api.GetAuthMethodResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	authMethod, err := s.repository.GetAuthMethod(ctx, req.Msg.AuthMethodId)
	if err != nil {
		return nil, err
	}

	if authMethod == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	return connect.NewResponse(&api.GetAuthMethodResponse{AuthMethod: &api.AuthMethod{
		Id:           authMethod.ID,
		Type:         authMethod.Type,
		Name:         authMethod.Name,
		Issuer:       authMethod.Issuer,
		ClientId:     authMethod.ClientId,
		ClientSecret: authMethod.ClientSecret,
	}}), nil
}

func (s *Service) CreateAuthMethod(ctx context.Context, req *connect.Request[api.CreateAuthMethodRequest]) (*connect.Response[api.CreateAuthMethodResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

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
		Id:           authMethod.ID,
		Type:         authMethod.Type,
		Name:         authMethod.Name,
		Issuer:       authMethod.Issuer,
		ClientId:     authMethod.ClientId,
		ClientSecret: authMethod.ClientSecret,
	}}), nil

}

func (s *Service) ListAuthMethods(ctx context.Context, _ *connect.Request[api.ListAuthMethodsRequest]) (*connect.Response[api.ListAuthMethodsResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

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

func (s *Service) DeleteAuthMethod(ctx context.Context, req *connect.Request[api.DeleteAuthMethodRequest]) (*connect.Response[api.DeleteAuthMethodResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	count, err := s.repository.CountMachinesByAuthMethod(ctx, req.Msg.AuthMethodId)
	if err != nil {
		return nil, err
	}

	if !req.Msg.Force && count > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("there are still machines authenticated using this method, number of machines: %d", count))
	}

	err = s.repository.Transaction(func(rp domain.Repository) error {

		if _, err := rp.DeleteMachinesByAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		if _, err := rp.DeleteAuthKeysByAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		if _, err := rp.DeleteApiKeysByAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		if _, err := rp.DeleteUsersByAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		if _, err := rp.DeleteAccountsByAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		if err := rp.DeleteAuthMethod(ctx, req.Msg.AuthMethodId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.brokerPool.SignalUpdate()

	return connect.NewResponse(&api.DeleteAuthMethodResponse{}), nil
}
