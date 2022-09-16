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
	principal := CurrentPrincipal(ctx)

	key, err := s.repository.GetAuthKey(ctx, req.Msg.AuthKeyId)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("auth key not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(key.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
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

func mapAuthKeysToApi(authKeys []domain.AuthKey) []*api.AuthKey {
	var result []*api.AuthKey

	for _, key := range authKeys {
		var expiresAt *timestamppb.Timestamp
		if key.ExpiresAt != nil {
			expiresAt = timestamppb.New(*key.ExpiresAt)
		}

		result = append(result, &api.AuthKey{
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
		})
	}

	return result
}

func (s *Service) ListAuthKeys(ctx context.Context, req *connect.Request[api.ListAuthKeysRequest]) (*connect.Response[api.ListAuthKeysResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	response := api.ListAuthKeysResponse{}

	if principal.IsSystemAdmin() {
		authKeys, err := s.repository.ListAuthKeys(ctx, req.Msg.TailnetId)
		if err != nil {
			return nil, err
		}

		response.AuthKeys = mapAuthKeysToApi(authKeys)
		return connect.NewResponse(&response), nil
	}

	if principal.User != nil {
		authKeys, err := s.repository.ListAuthKeysByTailnetAndUser(ctx, req.Msg.TailnetId, principal.User.ID)
		if err != nil {
			return nil, err
		}

		response.AuthKeys = mapAuthKeysToApi(authKeys)
		return connect.NewResponse(&response), nil
	}

	return connect.NewResponse(&response), nil
}

func (s *Service) CreateAuthKey(ctx context.Context, req *connect.Request[api.CreateAuthKeyRequest]) (*connect.Response[api.CreateAuthKeyResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if principal.User == nil && len(req.Msg.Tags) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one tag is required when creating an auth key"))
	}

	if err := domain.CheckTags(req.Msg.Tags); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	if principal.IsSystemAdmin() {
		if err := tailnet.ACLPolicy.CheckTags(req.Msg.Tags); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		if err := tailnet.ACLPolicy.CheckTagOwners(req.Msg.Tags, principal.User); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	var expiresAt *time.Time
	var expiresAtPb *timestamppb.Timestamp

	if req.Msg.Expiry != nil {
		duration := req.Msg.Expiry.AsDuration()
		e := time.Now().UTC().Add(duration)
		expiresAt = &e
		expiresAtPb = timestamppb.New(*expiresAt)
	}

	var user = principal.User
	if user == nil {
		u, _, err := s.repository.GetOrCreateServiceUser(ctx, tailnet)
		if err != nil {
			return nil, err
		}
		user = u
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
	principal := CurrentPrincipal(ctx)

	key, err := s.repository.GetAuthKey(ctx, req.Msg.AuthKeyId)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("auth key not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(key.UserID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	if _, err := s.repository.DeleteAuthKey(ctx, req.Msg.AuthKeyId); err != nil {
		return nil, err
	}
	return connect.NewResponse(&api.DeleteAuthKeyResponse{}), nil
}
