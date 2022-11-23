package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/errors"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"time"
)

func (s *Service) Authenticate(ctx context.Context, req *connect.Request[api.AuthenticateRequest], stream *connect.ServerStream[api.AuthenticateResponse]) error {
	if s.authProvider == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no authentication method available, contact your ionscale administrator for more information"))
	}

	key := util.RandStringBytes(8)
	authUrl := s.config.CreateUrl("/a/c/%s", key)

	session := &domain.AuthenticationRequest{
		Key:       key,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repository.SaveAuthenticationRequest(ctx, session); err != nil {
		return errors.Wrap(err, 0)
	}

	if err := stream.Send(&api.AuthenticateResponse{AuthUrl: authUrl}); err != nil {
		return errors.Wrap(err, 0)
	}

	notify := ctx.Done()
	tick := time.NewTicker(1 * time.Second)

	defer func() {
		tick.Stop()
		_ = s.repository.DeleteAuthenticationRequest(context.Background(), key)
	}()

	for {
		select {
		case <-tick.C:
			m, err := s.repository.GetAuthenticationRequest(ctx, key)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if m == nil {
				return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid authentication request"))
			}

			if len(m.Token) != 0 {
				if err := stream.Send(&api.AuthenticateResponse{Token: m.Token, TailnetId: m.TailnetID}); err != nil {
					return errors.Wrap(err, 0)
				}
				return nil
			}

			if len(m.Error) != 0 {
				return connect.NewError(connect.CodePermissionDenied, fmt.Errorf(m.Error))
			}

			if err := stream.Send(&api.AuthenticateResponse{AuthUrl: authUrl}); err != nil {
				return errors.Wrap(err, 0)
			}

		case <-notify:
			return nil
		}
	}
}
