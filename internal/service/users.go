package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/errors"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) ListUsers(ctx context.Context, req *connect.Request[api.ListUsersRequest]) (*connect.Response[api.ListUsersResponse], error) {
	principal := CurrentPrincipal(ctx)

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(tailnet.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	users, err := s.repository.ListUsers(ctx, tailnet.ID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
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

func (s *Service) DeleteUser(ctx context.Context, req *connect.Request[api.DeleteUserRequest]) (*connect.Response[api.DeleteUserResponse], error) {
	principal := CurrentPrincipal(ctx)

	if !principal.IsSystemAdmin() && principal.UserMatches(req.Msg.UserId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unable delete yourself"))
	}

	user, err := s.repository.GetUser(ctx, req.Msg.UserId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}

	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(user.TailnetID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	err = s.repository.Transaction(func(tx domain.Repository) error {
		if err := tx.DeleteMachineByUser(ctx, req.Msg.UserId); err != nil {
			return err
		}

		if err := tx.DeleteApiKeysByUser(ctx, req.Msg.UserId); err != nil {
			return err
		}

		if err := tx.DeleteAuthKeysByUser(ctx, req.Msg.UserId); err != nil {
			return err
		}

		if err := tx.DeleteUser(ctx, req.Msg.UserId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(user.TailnetID, &broker.Signal{})

	return connect.NewResponse(&api.DeleteUserResponse{}), nil
}
