package service

import (
	"context"
	"encoding/json"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetACLPolicy(ctx context.Context, req *api.GetACLPolicyRequest) (*api.GetACLPolicyResponse, error) {
	policy, err := s.repository.GetACLPolicy(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	marshal, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	return &api.GetACLPolicyResponse{Value: marshal}, nil
}

func (s *Service) SetACLPolicy(ctx context.Context, req *api.SetACLPolicyRequest) (*api.SetACLPolicyResponse, error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "tailnet does not exist")
	}

	var policy domain.ACLPolicy
	if err := json.Unmarshal(req.Value, &policy); err != nil {
		return nil, err
	}

	if err := s.repository.SetACLPolicy(ctx, tailnet.ID, &policy); err != nil {
		return nil, err
	}

	s.brokers(tailnet.ID).SignalACLUpdated()

	return &api.SetACLPolicyResponse{}, nil
}
