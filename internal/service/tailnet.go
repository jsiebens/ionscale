package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/jsiebens/ionscale/pkg/defaults"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func domainTailnetToApiTailnet(tailnet *domain.Tailnet) (*api.Tailnet, error) {
	t := &api.Tailnet{
		Id:                          tailnet.ID,
		Name:                        tailnet.Name,
		IamPolicy:                   new(api.IAMPolicy),
		AclPolicy:                   new(api.ACLPolicy),
		DnsConfig:                   domainDNSConfigToApiDNSConfig(tailnet),
		ServiceCollectionEnabled:    tailnet.ServiceCollectionEnabled,
		FileSharingEnabled:          tailnet.FileSharingEnabled,
		SshEnabled:                  tailnet.SSHEnabled,
		MachineAuthorizationEnabled: tailnet.MachineAuthorizationEnabled,
	}

	if err := mapping.CopyViaJson(tailnet.IAMPolicy, t.IamPolicy); err != nil {
		return nil, err
	}

	if err := mapping.CopyViaJson(tailnet.ACLPolicy, t.AclPolicy); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *Service) CreateTailnet(ctx context.Context, req *connect.Request[api.CreateTailnetRequest]) (*connect.Response[api.CreateTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	check, err := s.repository.GetTailnetByName(ctx, req.Msg.Name)
	if err != nil {
		return nil, logError(err)
	}
	if check != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tailnet with name '%s' already exists", req.Msg.Name))
	}

	if req.Msg.IamPolicy == nil {
		req.Msg.IamPolicy = defaults.DefaultIAMPolicy()
	}

	if req.Msg.AclPolicy == nil {
		req.Msg.AclPolicy = defaults.DefaultACLPolicy()
	}

	if req.Msg.DnsConfig == nil {
		req.Msg.DnsConfig = defaults.DefaultDNSConfig()
	}

	tailnet := &domain.Tailnet{
		ID:                          util.NextID(),
		Name:                        req.Msg.Name,
		IAMPolicy:                   domain.IAMPolicy{},
		ACLPolicy:                   domain.ACLPolicy{},
		DNSConfig:                   apiDNSConfigToDomainDNSConfig(req.Msg.DnsConfig),
		ServiceCollectionEnabled:    req.Msg.ServiceCollectionEnabled,
		FileSharingEnabled:          req.Msg.FileSharingEnabled,
		SSHEnabled:                  req.Msg.SshEnabled,
		MachineAuthorizationEnabled: req.Msg.MachineAuthorizationEnabled,
	}

	if err := validateIamPolicy(req.Msg.IamPolicy); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid iam policy: %w", err))
	}

	if err := mapping.CopyViaJson(req.Msg.IamPolicy, &tailnet.IAMPolicy); err != nil {
		return nil, logError(err)
	}

	if err := mapping.CopyViaJson(req.Msg.AclPolicy, &tailnet.ACLPolicy); err != nil {
		return nil, logError(err)
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	t, err := domainTailnetToApiTailnet(tailnet)
	if err != nil {
		return nil, logError(err)
	}

	resp := &api.CreateTailnetResponse{Tailnet: t}

	return connect.NewResponse(resp), nil
}

func (s *Service) UpdateTailnet(ctx context.Context, req *connect.Request[api.UpdateTailnetRequest]) (*connect.Response[api.UpdateTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if req.Msg.IamPolicy != nil {
		if err := validateIamPolicy(req.Msg.IamPolicy); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid iam policy: %w", err))
		}

		tailnet.IAMPolicy = domain.IAMPolicy{}
		if err := mapping.CopyViaJson(req.Msg.IamPolicy, &tailnet.IAMPolicy); err != nil {
			return nil, logError(err)
		}
	}

	if req.Msg.AclPolicy != nil {
		tailnet.ACLPolicy = domain.ACLPolicy{}
		if err := mapping.CopyViaJson(req.Msg.AclPolicy, &tailnet.ACLPolicy); err != nil {
			return nil, logError(err)
		}
	}

	if req.Msg.DnsConfig != nil {
		tailnet.DNSConfig = apiDNSConfigToDomainDNSConfig(req.Msg.DnsConfig)
	}

	tailnet.ServiceCollectionEnabled = req.Msg.ServiceCollectionEnabled
	tailnet.FileSharingEnabled = req.Msg.FileSharingEnabled
	tailnet.SSHEnabled = req.Msg.SshEnabled
	tailnet.MachineAuthorizationEnabled = req.Msg.MachineAuthorizationEnabled

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(tailnet.ID)

	t, err := domainTailnetToApiTailnet(tailnet)
	if err != nil {
		return nil, logError(err)
	}

	resp := &api.UpdateTailnetResponse{Tailnet: t}

	return connect.NewResponse(resp), nil
}

func (s *Service) GetTailnet(ctx context.Context, req *connect.Request[api.GetTailnetRequest]) (*connect.Response[api.GetTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.Id) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.Id)
	if err != nil {
		return nil, logError(err)
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	t, err := domainTailnetToApiTailnet(tailnet)
	if err != nil {
		return nil, logError(err)
	}

	return connect.NewResponse(&api.GetTailnetResponse{Tailnet: t}), nil
}

func (s *Service) ListTailnets(ctx context.Context, req *connect.Request[api.ListTailnetsRequest]) (*connect.Response[api.ListTailnetsResponse], error) {
	principal := CurrentPrincipal(ctx)

	resp := &api.ListTailnetsResponse{}

	if principal.IsSystemAdmin() {
		tailnets, err := s.repository.ListTailnets(ctx)
		if err != nil {
			return nil, logError(err)
		}
		for _, t := range tailnets {
			gt := api.Tailnet{Id: t.ID, Name: t.Name}
			resp.Tailnet = append(resp.Tailnet, &gt)
		}
	}

	if principal.User != nil {
		tailnet, err := s.repository.GetTailnet(ctx, principal.User.TailnetID)
		if err != nil {
			return nil, logError(err)
		}
		gt := api.Tailnet{Id: tailnet.ID, Name: tailnet.Name}
		resp.Tailnet = append(resp.Tailnet, &gt)
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) DeleteTailnet(ctx context.Context, req *connect.Request[api.DeleteTailnetRequest]) (*connect.Response[api.DeleteTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	count, err := s.repository.CountMachineByTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}

	if !req.Msg.Force && count > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tailnet is not empty, number of machines: %d", count))
	}

	err = s.repository.Transaction(func(tx domain.Repository) error {
		if err := tx.DeleteMachineByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteApiKeysByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteAuthKeysByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteUsersByTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		if err := tx.DeleteTailnet(ctx, req.Msg.TailnetId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(req.Msg.TailnetId)

	return connect.NewResponse(&api.DeleteTailnetResponse{}), nil
}

func (s *Service) SetDERPMap(ctx context.Context, req *connect.Request[api.SetDERPMapRequest]) (*connect.Response[api.SetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	derpMap := tailcfg.DERPMap{}
	if err := json.Unmarshal(req.Msg.Value, &derpMap); err != nil {
		return nil, logError(err)
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DERPMap = domain.DERPMap{
		Checksum: util.Checksum(&derpMap),
		DERPMap:  derpMap,
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(tailnet.ID)

	raw, err := json.Marshal(derpMap)
	if err != nil {
		return nil, logError(err)
	}

	return connect.NewResponse(&api.SetDERPMapResponse{Value: raw}), nil
}

func (s *Service) ResetDERPMap(ctx context.Context, req *connect.Request[api.ResetDERPMapRequest]) (*connect.Response[api.ResetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DERPMap = domain.DERPMap{}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(tailnet.ID)

	return connect.NewResponse(&api.ResetDERPMapResponse{}), nil
}

func (s *Service) GetDERPMap(ctx context.Context, req *connect.Request[api.GetDERPMapRequest]) (*connect.Response[api.GetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	derpMap, err := tailnet.GetDERPMap(ctx, s.repository)
	if err != nil {
		return nil, logError(err)
	}

	raw, err := json.Marshal(derpMap.DERPMap)
	if err != nil {
		return nil, logError(err)
	}

	return connect.NewResponse(&api.GetDERPMapResponse{Value: raw}), nil
}

func (s *Service) EnableFileSharing(ctx context.Context, req *connect.Request[api.EnableFileSharingRequest]) (*connect.Response[api.EnableFileSharingResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.EnableFileSharingResponse{}), nil
}

func (s *Service) DisableFileSharing(ctx context.Context, req *connect.Request[api.DisableFileSharingRequest]) (*connect.Response[api.DisableFileSharingResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.DisableFileSharingResponse{}), nil
}

func (s *Service) EnableServiceCollection(ctx context.Context, req *connect.Request[api.EnableServiceCollectionRequest]) (*connect.Response[api.EnableServiceCollectionResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.EnableServiceCollectionResponse{}), nil
}

func (s *Service) DisableServiceCollection(ctx context.Context, req *connect.Request[api.DisableServiceCollectionRequest]) (*connect.Response[api.DisableServiceCollectionResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.DisableServiceCollectionResponse{}), nil
}

func (s *Service) EnableSSH(ctx context.Context, req *connect.Request[api.EnableSSHRequest]) (*connect.Response[api.EnableSSHResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.SSHEnabled {
		tailnet.SSHEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.EnableSSHResponse{}), nil
}

func (s *Service) DisableSSH(ctx context.Context, req *connect.Request[api.DisableSSHRequest]) (*connect.Response[api.DisableSSHResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.SSHEnabled {
		tailnet.SSHEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}

		s.sessionManager.NotifyAll(tailnet.ID)
	}

	return connect.NewResponse(&api.DisableSSHResponse{}), nil
}

func (s *Service) EnableMachineAuthorization(ctx context.Context, req *connect.Request[api.EnableMachineAuthorizationRequest]) (*connect.Response[api.EnableMachineAuthorizationResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.MachineAuthorizationEnabled {
		tailnet.MachineAuthorizationEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}
	}

	return connect.NewResponse(&api.EnableMachineAuthorizationResponse{}), nil
}

func (s *Service) DisableMachineAuthorization(ctx context.Context, req *connect.Request[api.DisableMachineAuthorizationRequest]) (*connect.Response[api.DisableMachineAuthorizationResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.MachineAuthorizationEnabled {
		tailnet.MachineAuthorizationEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, logError(err)
		}
	}

	return connect.NewResponse(&api.DisableMachineAuthorizationResponse{}), nil
}
