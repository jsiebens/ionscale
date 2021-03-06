package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetIAMPolicy(ctx context.Context, req *connect.Request[api.GetIAMPolicyRequest]) (*connect.Response[api.GetIAMPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet does not exist"))
	}

	policy := &api.IAMPolicy{
		Subs:    tailnet.IAMPolicy.Subs,
		Emails:  tailnet.IAMPolicy.Emails,
		Filters: tailnet.IAMPolicy.Filters,
		Roles:   domainRolesMapToApiRolesMap(tailnet.IAMPolicy.Roles),
	}

	return connect.NewResponse(&api.GetIAMPolicyResponse{Policy: policy}), nil
}

func (s *Service) SetIAMPolicy(ctx context.Context, req *connect.Request[api.SetIAMPolicyRequest]) (*connect.Response[api.SetIAMPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet does not exist"))
	}

	tailnet.IAMPolicy = domain.IAMPolicy{
		Subs:    req.Msg.Policy.Subs,
		Emails:  req.Msg.Policy.Emails,
		Filters: req.Msg.Policy.Filters,
		Roles:   apiRolesMapToDomainRolesMap(req.Msg.Policy.Roles),
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.SetIAMPolicyResponse{}), nil
}

func apiRolesMapToDomainRolesMap(values map[string]string) map[string]domain.UserRole {
	var result = map[string]domain.UserRole{}
	for k, v := range values {
		result[k] = domain.UserRole(v)
	}
	return result
}

func domainRolesMapToApiRolesMap(values map[string]domain.UserRole) map[string]string {
	var result = map[string]string{}
	for k, v := range values {
		result[k] = string(v)
	}
	return result
}
