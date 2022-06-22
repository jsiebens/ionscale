package service

import (
	"context"
	"errors"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) ListUsers(ctx context.Context, req *connect.Request[api.ListUsersRequest]) (*connect.Response[api.ListUsersResponse], error) {
	principal := CurrentPrincipal(ctx)

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(tailnet.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	users, err := s.repository.ListUsers(ctx, tailnet.ID)
	if err != nil {
		return nil, err
	}

	resp := &api.ListUsersResponse{}
	for _, u := range users {
		resp.Users = append(resp.Users, &api.User{
			Id:   u.ID,
			Name: u.Name,
			Role: string(tailnet.IAMPolicy.GetRole(u)),
		})
	}

	return connect.NewResponse(resp), nil
}
