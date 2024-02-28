package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func (s *Service) GetDefaultDERPMap(ctx context.Context, _ *connect.Request[api.GetDefaultDERPMapRequest]) (*connect.Response[api.GetDefaultDERPMapResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	dm := domain.GetDefaultDERPMap()

	raw, err := json.Marshal(dm.DERPMap)
	if err != nil {
		return nil, logError(err)
	}

	return connect.NewResponse(&api.GetDefaultDERPMapResponse{Value: raw}), nil
}
