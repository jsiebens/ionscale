package service

import (
	"context"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/auth"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/version"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func NewService(config *config.Config, authProvider auth.Provider, dnsProvider dns.Provider, repository domain.Repository, sessionManager core.PollMapSessionManager) *Service {
	return &Service{
		config:         config,
		authProvider:   authProvider,
		dnsProvider:    dnsProvider,
		repository:     repository,
		sessionManager: sessionManager,
	}
}

type Service struct {
	config         *config.Config
	authProvider   auth.Provider
	dnsProvider    dns.Provider
	repository     domain.Repository
	sessionManager core.PollMapSessionManager
}

func (s *Service) GetVersion(_ context.Context, _ *connect.Request[api.GetVersionRequest]) (*connect.Response[api.GetVersionResponse], error) {
	v, revision := version.GetReleaseInfo()
	return connect.NewResponse(&api.GetVersionResponse{
		Version:  v,
		Revision: revision,
	}), nil
}
