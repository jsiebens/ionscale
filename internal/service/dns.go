package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/util/dnsname"
)

func (s *Service) GetDNSConfig(ctx context.Context, req *connect.Request[api.GetDNSConfigRequest]) (*connect.Response[api.GetDNSConfigResponse], error) {
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

	config := tailnet.DNSConfig
	tailnetDomain := dnsname.SanitizeHostname(tailnet.Name)

	resp := &api.GetDNSConfigResponse{
		Config: &api.DNSConfig{
			MagicDns:         config.MagicDNS,
			MagicDnsSuffix:   fmt.Sprintf("%s.%s", tailnetDomain, mapping.NetworkMagicDNSSuffix),
			OverrideLocalDns: config.OverrideLocalDNS,
			Nameservers:      config.Nameservers,
			Routes:           domainRoutesToApiRoutes(config.Routes),
		},
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) SetDNSConfig(ctx context.Context, req *connect.Request[api.SetDNSConfigRequest]) (*connect.Response[api.SetDNSConfigResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

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

	tailnet.DNSConfig = domain.DNSConfig{
		MagicDNS:         dnsConfig.MagicDns,
		OverrideLocalDNS: dnsConfig.OverrideLocalDns,
		Nameservers:      dnsConfig.Nameservers,
		Routes:           apiRoutesToDomainRoutes(dnsConfig.Routes),
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, err
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{DNSUpdated: true})

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
