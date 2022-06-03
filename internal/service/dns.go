package service

import (
	"context"
	"errors"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetDNSConfig(ctx context.Context, req *connect.Request[api.GetDNSConfigRequest]) (*connect.Response[api.GetDNSConfigResponse], error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	config, err := s.repository.GetDNSConfig(ctx, tailnet.ID)
	if err != nil {
		return nil, err
	}

	resp := &api.GetDNSConfigResponse{
		Config: &api.DNSConfig{
			MagicDns:         config.MagicDNS,
			OverrideLocalDns: config.OverrideLocalDNS,
			Nameservers:      config.Nameservers,
			Routes:           domainRoutesToApiRoutes(config.Routes),
		},
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) SetDNSConfig(ctx context.Context, req *connect.Request[api.SetDNSConfigRequest]) (*connect.Response[api.SetDNSConfigResponse], error) {
	dnsConfig := req.Msg.Config

	if dnsConfig.MagicDns && len(dnsConfig.Nameservers) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one global nameserver is required when enabling magic dns"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	config := domain.DNSConfig{
		MagicDNS:         dnsConfig.MagicDns,
		OverrideLocalDNS: dnsConfig.OverrideLocalDns,
		Nameservers:      dnsConfig.Nameservers,
		Routes:           apiRoutesToDomainRoutes(dnsConfig.Routes),
	}

	if err := s.repository.SetDNSConfig(ctx, tailnet.ID, &config); err != nil {
		return nil, err
	}

	s.brokers(tailnet.ID).SignalDNSUpdated()

	resp := &api.SetDNSConfigResponse{Config: dnsConfig}

	return connect.NewResponse(resp), nil
}

func domainRoutesToApiRoutes(routes map[string][]string) map[string]*api.Routes {
	var result = map[string]*api.Routes{}
	for k, v := range routes {
		result[k] = &api.Routes{Routes: v}
	}
	return result
}

func apiRoutesToDomainRoutes(routes map[string]*api.Routes) map[string][]string {
	var result = map[string][]string{}
	for k, v := range routes {
		result[k] = v.Routes
	}
	return result
}
