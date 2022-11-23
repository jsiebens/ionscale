package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/errors"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) CreateTailnet(ctx context.Context, req *connect.Request[api.CreateTailnetRequest]) (*connect.Response[api.CreateTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	name := req.Msg.Name
	iamPolicy := domain.IAMPolicy{}

	if req.Msg.IamPolicy != nil {
		iamPolicy.Subs = req.Msg.IamPolicy.Subs
		iamPolicy.Emails = req.Msg.IamPolicy.Emails
		iamPolicy.Filters = req.Msg.IamPolicy.Filters
		iamPolicy.Roles = apiRolesMapToDomainRolesMap(req.Msg.IamPolicy.Roles)
	}

	tailnet := &domain.Tailnet{
		ID:        util.NextID(),
		Name:      name,
		IAMPolicy: iamPolicy,
		ACLPolicy: domain.DefaultPolicy(),
		DNSConfig: domain.DNSConfig{MagicDNS: true},
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	resp := &api.CreateTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}

	return connect.NewResponse(resp), nil
}

func (s *Service) GetTailnet(ctx context.Context, req *connect.Request[api.GetTailnetRequest]) (*connect.Response[api.GetTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.Id) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.Id)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	return connect.NewResponse(&api.GetTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}), nil
}

func (s *Service) ListTailnets(ctx context.Context, req *connect.Request[api.ListTailnetsRequest]) (*connect.Response[api.ListTailnetsResponse], error) {
	principal := CurrentPrincipal(ctx)

	resp := &api.ListTailnetsResponse{}

	if principal.IsSystemAdmin() {
		tailnets, err := s.repository.ListTailnets(ctx)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		for _, t := range tailnets {
			gt := api.Tailnet{Id: t.ID, Name: t.Name}
			resp.Tailnet = append(resp.Tailnet, &gt)
		}
	}

	if principal.User != nil {
		tailnet, err := s.repository.GetTailnet(ctx, principal.User.TailnetID)
		if err != nil {
			return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(req.Msg.TailnetId, &broker.Signal{})

	return connect.NewResponse(&api.DeleteTailnetResponse{}), nil
}

func (s *Service) SetDERPMap(ctx context.Context, req *connect.Request[api.SetDERPMapRequest]) (*connect.Response[api.SetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	derpMap := tailcfg.DERPMap{}
	if err := json.Unmarshal(req.Msg.Value, &derpMap); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DERPMap = domain.DERPMap{
		Checksum: util.Checksum(&derpMap),
		DERPMap:  derpMap,
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{})

	raw, err := json.Marshal(derpMap)
	if err != nil {
		return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	tailnet.DERPMap = domain.DERPMap{}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, errors.Wrap(err, 0)
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{})

	return connect.NewResponse(&api.ResetDERPMapResponse{}), nil
}

func (s *Service) GetDERPMap(ctx context.Context, req *connect.Request[api.GetDERPMapRequest]) (*connect.Response[api.GetDERPMapResponse], error) {
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

	derpMap, err := tailnet.GetDERPMap(ctx, s.repository)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	raw, err := json.Marshal(derpMap.DERPMap)
	if err != nil {
		return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.SSHEnabled {
		tailnet.SSHEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.SSHEnabled {
		tailnet.SSHEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if !tailnet.MachineAuthorizationEnabled {
		tailnet.MachineAuthorizationEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
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
		return nil, errors.Wrap(err, 0)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	if tailnet.MachineAuthorizationEnabled {
		tailnet.MachineAuthorizationEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	return connect.NewResponse(&api.DisableMachineAuthorizationResponse{}), nil
}
