package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) CreateTailnet(ctx context.Context, req *connect.Request[api.CreateTailnetRequest]) (*connect.Response[api.CreateTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	name := req.Msg.Name
	iamPolicy := domain.IAMPolicy{
		Subs:    req.Msg.IamPolicy.Subs,
		Emails:  req.Msg.IamPolicy.Emails,
		Filters: req.Msg.IamPolicy.Filters,
		Roles:   apiRolesMapToDomainRolesMap(req.Msg.IamPolicy.Roles),
	}

	tailnet, created, err := s.repository.GetOrCreateTailnet(ctx, name, iamPolicy)
	if err != nil {
		return nil, err
	}

	if !created {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tailnet already exists"))
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
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}

	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	return connect.NewResponse(&api.GetTailnetResponse{Tailnet: &api.Tailnet{
		Id:   tailnet.ID,
		Name: tailnet.Name,
	}}), nil
}

func (s *Service) ListTailnets(ctx context.Context, req *connect.Request[api.ListTailnetRequest]) (*connect.Response[api.ListTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)

	resp := &api.ListTailnetResponse{}

	if principal.IsSystemAdmin() {
		tailnets, err := s.repository.ListTailnets(ctx)
		if err != nil {
			return nil, err
		}
		for _, t := range tailnets {
			gt := api.Tailnet{Id: t.ID, Name: t.Name}
			resp.Tailnet = append(resp.Tailnet, &gt)
		}
	}

	if principal.User != nil {
		tailnet, err := s.repository.GetTailnet(ctx, principal.User.TailnetID)
		if err != nil {
			return nil, err
		}
		gt := api.Tailnet{Id: tailnet.ID, Name: tailnet.Name}
		resp.Tailnet = append(resp.Tailnet, &gt)
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) DeleteTailnet(ctx context.Context, req *connect.Request[api.DeleteTailnetRequest]) (*connect.Response[api.DeleteTailnetResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	count, err := s.repository.CountMachineByTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	s.pubsub.Publish(req.Msg.TailnetId, &broker.Signal{})

	return connect.NewResponse(&api.DeleteTailnetResponse{}), nil
}

func (s *Service) SetDERPMap(ctx context.Context, req *connect.Request[api.SetDERPMapRequest]) (*connect.Response[api.SetDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("permission denied"))
	}

	derpMap := tailcfg.DERPMap{}
	if err := json.Unmarshal(req.Msg.Value, &derpMap); err != nil {
		return nil, err
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, err
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tailnet not found"))
	}

	tailnet.DERPMap = domain.DERPMap{
		Checksum: util.Checksum(&derpMap),
		DERPMap:  derpMap,
	}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, err
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{})

	raw, err := json.Marshal(derpMap)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.SetDERPMapResponse{Value: raw}), nil
}

func (s *Service) ResetDERPMap(ctx context.Context, req *connect.Request[api.ResetDERPMapRequest]) (*connect.Response[api.ResetDERPMapResponse], error) {
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

	tailnet.DERPMap = domain.DERPMap{}

	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, err
	}

	s.pubsub.Publish(tailnet.ID, &broker.Signal{})

	return connect.NewResponse(&api.ResetDERPMapResponse{}), nil
}

func (s *Service) GetDERPMap(ctx context.Context, req *connect.Request[api.GetDERPMapRequest]) (*connect.Response[api.GetDERPMapResponse], error) {
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

	derpMap, err := tailnet.GetDERPMap(ctx, s.repository)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(derpMap.DERPMap)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.GetDERPMapResponse{Value: raw}), nil
}

func (s *Service) EnabledFileSharing(ctx context.Context, req *connect.Request[api.EnableFileSharingRequest]) (*connect.Response[api.EnableFileSharingResponse], error) {
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

	if !tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, err
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.EnableFileSharingResponse{}), nil
}

func (s *Service) DisableFileSharing(ctx context.Context, req *connect.Request[api.DisableFileSharingRequest]) (*connect.Response[api.DisableFileSharingResponse], error) {
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

	if tailnet.FileSharingEnabled {
		tailnet.FileSharingEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, err
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.DisableFileSharingResponse{}), nil
}

func (s *Service) EnabledServiceCollection(ctx context.Context, req *connect.Request[api.EnableServiceCollectionRequest]) (*connect.Response[api.EnableServiceCollectionResponse], error) {
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

	if !tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = true
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, err
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.EnableServiceCollectionResponse{}), nil
}

func (s *Service) DisableServiceCollection(ctx context.Context, req *connect.Request[api.DisableServiceCollectionRequest]) (*connect.Response[api.DisableServiceCollectionResponse], error) {
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

	if tailnet.ServiceCollectionEnabled {
		tailnet.ServiceCollectionEnabled = false
		if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
			return nil, err
		}

		s.pubsub.Publish(tailnet.ID, &broker.Signal{})
	}

	return connect.NewResponse(&api.DisableServiceCollectionResponse{}), nil
}
