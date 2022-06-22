package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetACLPolicy(ctx context.Context, req *connect.Request[api.GetACLPolicyRequest]) (*connect.Response[api.GetACLPolicyResponse], error) {
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

	var policy api.ACLPolicy
	if err := mapping.CopyViaJson(&tailnet.ACLPolicy, &policy); err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetACLPolicyResponse{Policy: &policy}), nil
}

func (s *Service) SetACLPolicy(ctx context.Context, req *connect.Request[api.SetACLPolicyRequest]) (*connect.Response[api.SetACLPolicyResponse], error) {
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

	var policy domain.ACLPolicy
	if err := mapping.CopyViaJson(req.Msg.Policy, &policy); err != nil {
		return nil, err
	}

	tailnet.ACLPolicy = policy
	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, err
	}

	s.brokers(tailnet.ID).SignalACLUpdated()

	return connect.NewResponse(&api.SetACLPolicyResponse{}), nil
}
