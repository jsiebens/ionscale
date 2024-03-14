package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetIAMPolicy(ctx context.Context, req *connect.Request[api.GetIAMPolicyRequest]) (*connect.Response[api.GetIAMPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet does not exist"))
	}

	return connect.NewResponse(&api.GetIAMPolicyResponse{Policy: tailnet.IAMPolicy.String()}), nil
}

func (s *Service) SetIAMPolicy(ctx context.Context, req *connect.Request[api.SetIAMPolicyRequest]) (*connect.Response[api.SetIAMPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet does not exist"))
	}

	newPolicy, err := domain.ParseHuJson[domain.IAMPolicy](req.Msg.Policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid iam policy: %w", err))
	}

	if err := validateIamPolicy(newPolicy.Get()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid iam policy: %w", err))
	}

	oldPolicy := tailnet.IAMPolicy
	if oldPolicy.Equal(newPolicy) {
		return connect.NewResponse(&api.SetIAMPolicyResponse{}), nil
	}

	tailnet.IAMPolicy = *newPolicy

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	return connect.NewResponse(&api.SetIAMPolicyResponse{}), nil
}
