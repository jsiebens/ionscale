package service

import (
	"context"
	"errors"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func (s *Service) GetAuthKey(ctx context.Context, req *connect.Request[api.GetAuthKeyRequest]) (*connect.Response[api.GetAuthKeyResponse], error) {
	key, err := s.repository.GetAuthKey(ctx, req.Msg.AuthKeyId)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("auth key not found"))
	}

	var expiresAt *timestamppb.Timestamp
	if key.ExpiresAt != nil {
		expiresAt = timestamppb.New(*key.ExpiresAt)
	}

	return connect.NewResponse(&api.GetAuthKeyResponse{AuthKey: &api.AuthKey{
		Id:        key.ID,
		Key:       key.Key,
		Ephemeral: key.Ephemeral,
		Tags:      key.Tags,
		CreatedAt: timestamppb.New(key.CreatedAt),
		ExpiresAt: expiresAt,
		Tailnet: &api.Ref{
			Id:   key.Tailnet.ID,
			Name: key.Tailnet.Name,
		},
	}}), nil
}

func (s *Service) ListAuthKeys(ctx context.Context, req *connect.Request[api.ListAuthKeysRequest]) (*connect.Response[api.ListAuthKeysResponse], error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	authKeys, err := s.repository.ListAuthKeys(ctx, req.Msg.TailnetId)
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

	return connect.NewResponse(&response), nil
}

func (s *Service) CreateAuthKey(ctx context.Context, req *connect.Request[api.CreateAuthKeyRequest]) (*connect.Response[api.CreateAuthKeyResponse], error) {
	if len(req.Msg.Tags) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one tag is required when creating an auth key"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	var expiresAt *time.Time
	var expiresAtPb *timestamppb.Timestamp

	if req.Msg.Expiry != nil {
		duration := req.Msg.Expiry.AsDuration()
		e := time.Now().UTC().Add(duration)
		expiresAt = &e
		expiresAtPb = timestamppb.New(*expiresAt)
	}

	user, _, err := s.repository.GetOrCreateServiceUser(ctx, tailnet)
	if err != nil {
		return nil, err
	}

	tags := domain.SanitizeTags(req.Msg.Tags)

	v, authKey := domain.CreateAuthKey(tailnet, user, req.Msg.Ephemeral, tags, expiresAt)

	if err := s.repository.SaveAuthKey(ctx, authKey); err != nil {
		return nil, err
	}

	response := api.CreateAuthKeyResponse{
		Value: v,
		AuthKey: &api.AuthKey{
			Id:        authKey.ID,
			Key:       authKey.Key,
			Ephemeral: authKey.Ephemeral,
			Tags:      authKey.Tags,
			CreatedAt: timestamppb.New(authKey.CreatedAt),
			ExpiresAt: expiresAtPb,
			Tailnet: &api.Ref{
				Id:   tailnet.ID,
				Name: tailnet.Name,
			},
		}}

	return connect.NewResponse(&response), nil
}

func (s *Service) DeleteAuthKey(ctx context.Context, req *connect.Request[api.DeleteAuthKeyRequest]) (*connect.Response[api.DeleteAuthKeyResponse], error) {
	if _, err := s.repository.DeleteAuthKey(ctx, req.Msg.AuthKeyId); err != nil {
		return nil, err
	}
	return connect.NewResponse(&api.DeleteAuthKeyResponse{}), nil
}
