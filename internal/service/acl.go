package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetACLPolicy(ctx context.Context, req *connect.Request[api.GetACLPolicyRequest]) (*connect.Response[api.GetACLPolicyResponse], error) {
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

	return connect.NewResponse(&api.GetACLPolicyResponse{Policy: tailnet.ACLPolicy.String()}), nil
}

func (s *Service) SetACLPolicy(ctx context.Context, req *connect.Request[api.SetACLPolicyRequest]) (*connect.Response[api.SetACLPolicyResponse], error) {
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

	newPolicy, err := domain.ParseHuJson[domain.ACLPolicy](req.Msg.Policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid acl policy: %w", err))
	}

	oldPolicy := tailnet.ACLPolicy
	if oldPolicy.Equal(newPolicy) {
		return connect.NewResponse(&api.SetACLPolicyResponse{}), nil
	}

	tailnet.ACLPolicy = *newPolicy

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(tailnet.ID)

	return connect.NewResponse(&api.SetACLPolicyResponse{}), nil
}
