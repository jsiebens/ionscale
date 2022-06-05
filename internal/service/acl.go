package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetACLPolicy(ctx context.Context, req *connect.Request[api.GetACLPolicyRequest]) (*connect.Response[api.GetACLPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.TailnetMatches(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet does not exist"))
	}

	policy, err := s.repository.GetACLPolicy(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	marshal, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetACLPolicyResponse{Value: marshal}), nil
}

func (s *Service) SetACLPolicy(ctx context.Context, req *connect.Request[api.SetACLPolicyRequest]) (*connect.Response[api.SetACLPolicyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
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
	if err := json.Unmarshal(req.Msg.Value, &policy); err != nil {
		return nil, err
	}

	if err := s.repository.SetACLPolicy(ctx, tailnet.ID, &policy); err != nil {
		return nil, err
	}

	s.brokers(tailnet.ID).SignalACLUpdated()

	return connect.NewResponse(&api.SetACLPolicyResponse{}), nil
}
