package service

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/errors"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetDNSConfig(ctx context.Context, req *connect.Request[api.GetDNSConfigRequest]) (*connect.Response[api.GetDNSConfigResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	dnsConfig := tailnet.DNSConfig
	tailnetDomain := domain.SanitizeTailnetName(tailnet.Name)

	resp := &api.GetDNSConfigResponse{
		Config: &api.DNSConfig{
			MagicDns:         dnsConfig.MagicDNS,
			MagicDnsSuffix:   fmt.Sprintf("%s.%s", tailnetDomain, config.MagicDNSSuffix()),
			OverrideLocalDns: dnsConfig.OverrideLocalDNS,
			Nameservers:      dnsConfig.Nameservers,
			Routes:           domainRoutesToApiRoutes(dnsConfig.Routes),
		},
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) SetDNSConfig(ctx context.Context, req *connect.Request[api.SetDNSConfigRequest]) (*connect.Response[api.SetDNSConfigResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	dnsConfig := req.Msg.Config

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DNSConfig = domain.DNSConfig{
		MagicDNS:         dnsConfig.MagicDns,
		OverrideLocalDNS: dnsConfig.OverrideLocalDns,
		Nameservers:      dnsConfig.Nameservers,
		Routes:           apiRoutesToDomainRoutes(dnsConfig.Routes),
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{DNSUpdated: true})

	resp := &api.SetDNSConfigResponse{Config: dnsConfig}

	return connect.NewResponse(resp), nil
}

func (s *Service) EnableHttpsCertificates(ctx context.Context, req *connect.Request[api.EnableHttpsCertificatesRequest]) (*connect.Response[api.EnableHttpsCertificatesResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.DNSConfig.MagicDNS {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("MagicDNS must be enabled for this tailnet"))
	}

	tailnet.DNSConfig.HttpsCertsEnabled = true
	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{DNSUpdated: true})

	return connect.NewResponse(&api.EnableHttpsCertificatesResponse{}), nil
}

func (s *Service) DisableHttpsCertificates(ctx context.Context, req *connect.Request[api.DisableHttpsCertificatesRequest]) (*connect.Response[api.DisableHttpsCertificatesResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DNSConfig.HttpsCertsEnabled = false

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{DNSUpdated: true})

	return connect.NewResponse(&api.DisableHttpsCertificatesResponse{}), nil
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
