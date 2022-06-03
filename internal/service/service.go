package service

import (
	"context"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/version"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
)

func NewService(repository domain.Repository, brokerPool *broker.BrokerPool) *Service {
	return &Service{
		repository: repository,
		brokerPool: brokerPool,
	}
}

type Service struct {
	repository domain.Repository
	brokerPool *broker.BrokerPool
}

func (s *Service) brokers(tailnetID uint64) broker.Broker {
	return s.brokerPool.Get(tailnetID)
}

func (s *Service) GetVersion(_ context.Context, _ *connect.Request[api.GetVersionRequest]) (*connect.Response[api.GetVersionResponse], error) {
	v, revision := version.GetReleaseInfo()
	return connect.NewResponse(&api.GetVersionResponse{
		Version:  v,
		Revision: revision,
	}), nil
}
