package service

import (
	"context"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetDNSConfig(ctx context.Context, req *api.GetDNSConfigRequest) (*api.GetDNSConfigResponse, error) {
	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "tailnet does not exist")
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

	return resp, nil
}

func (s *Service) SetDNSConfig(ctx context.Context, req *api.SetDNSConfigRequest) (*api.SetDNSConfigResponse, error) {
	dnsConfig := req.Config

	if dnsConfig.MagicDns && len(dnsConfig.Nameservers) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one global nameserver is required when enabling magic dns")
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, status.Error(codes.NotFound, "tailnet does not exist")
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

	return resp, nil
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
