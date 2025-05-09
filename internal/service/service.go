package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/hashicorp/go-bexpr/grammar"
	"github.com/hashicorp/go-multierror"
	"github.com/jsiebens/ionscale/internal/auth"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/version"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func NewService(config *config.Config, authProvider auth.Provider, repository domain.Repository, sessionManager core.PollMapSessionManager) *Service {
	return &Service{
		config:         config,
		authProvider:   authProvider,
		repository:     repository,
		sessionManager: sessionManager,
	}
}

type Service struct {
	config               *config.Config
	authProvider         auth.Provider
	repository           domain.Repository
	sessionManager       core.PollMapSessionManager
	dnsProviderAvailable bool
}

func (s *Service) GetVersion(_ context.Context, _ *connect.Request[api.GetVersionRequest]) (*connect.Response[api.GetVersionResponse], error) {
	v, revision := version.GetReleaseInfo()
	return connect.NewResponse(&api.GetVersionResponse{
		Version:  v,
		Revision: revision,
	}), nil
}

func validateIamPolicy(p *domain.IAMPolicy) error {
	var mErr *multierror.Error
	for i, exp := range p.Filters {
		if _, err := grammar.Parse(fmt.Sprintf("filter %d", i), []byte(exp)); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}
	return mErr.ErrorOrNil()
}
