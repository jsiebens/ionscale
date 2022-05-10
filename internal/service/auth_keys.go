package service

import (
	"context"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func (s *Service) ListAuthKeys(ctx context.Context, req *api.ListAuthKeysRequest) (*api.ListAuthKeysResponse, error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "")
	}

	authKeys, err := s.repository.ListAuthKeys(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	response := api.ListAuthKeysResponse{}

	for _, key := range authKeys {
		var expiresAt *timestamppb.Timestamp
		if key.ExpiresAt != nil {
			expiresAt = timestamppb.New(*key.ExpiresAt)
		}

		response.AuthKeys = append(response.AuthKeys, &api.AuthKey{
			Id:        key.ID,
			Key:       key.Key,
			Ephemeral: key.Ephemeral,
			Tags:      key.Tags,
			CreatedAt: timestamppb.New(key.CreatedAt),
			ExpiresAt: expiresAt,
			Tailnet: &api.Ref{
				Id:   tailnet.ID,
				Name: tailnet.Name,
			},
		})
	}

	return &response, nil
}

func (s *Service) CreateAuthKey(ctx context.Context, req *api.CreateAuthKeyRequest) (*api.CreateAuthKeyResponse, error) {
	if len(req.Tags) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one tag is required when creating an auth key")
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "")
	}

	var expiresAt *time.Time
	var expiresAtPb *timestamppb.Timestamp

	if req.Expiry != nil {
		duration := req.Expiry.AsDuration()
		e := time.Now().UTC().Add(duration)
		expiresAt = &e
		expiresAtPb = timestamppb.New(*expiresAt)
	}

	user, _, err := s.repository.GetOrCreateServiceUser(ctx, tailnet)
	if err != nil {
		return nil, err
	}

	tags := domain.SanitizeTags(req.Tags)

	v, authKey := domain.CreateAuthKey(tailnet, user, req.Ephemeral, tags, expiresAt)

	if err := s.repository.SaveAuthKey(ctx, authKey); err != nil {
		return nil, err
	}

	response := api.CreateAuthKeyResponse{
		Value: v,
		AuthKey: &api.AuthKey{
			Id:        authKey.ID,
			Key:       authKey.Key,
			Ephemeral: authKey.Ephemeral,
			CreatedAt: timestamppb.New(authKey.CreatedAt),
			ExpiresAt: expiresAtPb,
			Tailnet: &api.Ref{
				Id:   tailnet.ID,
				Name: tailnet.Name,
			},
		}}

	return &response, nil
}

func (s *Service) DeleteAuthKey(ctx context.Context, req *api.DeleteAuthKeyRequest) (*api.DeleteAuthKeyResponse, error) {
	if _, err := s.repository.DeleteAuthKey(ctx, req.AuthKeyId); err != nil {
		return nil, err
	}
	return &api.DeleteAuthKeyResponse{}, nil
}
