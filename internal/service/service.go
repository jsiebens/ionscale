package service

import (
	"context"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/token"
	"github.com/jsiebens/ionscale/internal/version"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"tailscale.com/types/key"
)

var (
	errMissingMetadata = status.Error(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Error(codes.Unauthenticated, "invalid token")
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

func (s *Service) GetVersion(ctx context.Context, req *api.GetVersionRequest) (*api.GetVersionResponse, error) {
	v, revision := version.GetReleaseInfo()
	return &api.GetVersionResponse{
		Version:  v,
		Revision: revision,
	}, nil
}

func UnaryServerTokenAuth(systemAdminKey key.MachinePrivate) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if strings.HasSuffix(info.FullMethod, "/GetVersion") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errMissingMetadata
		}

		// The keys within metadata.MD are normalized to lowercase.
		// See: https://godoc.org/google.golang.org/grpc/metadata#New
		valid := validateAuthorizationToken(systemAdminKey, md["authorization"])

		if valid {
			return handler(ctx, req)
		}

		return nil, errInvalidToken
	}
}

func validateAuthorizationToken(systemAdminKey key.MachinePrivate, authorization []string) bool {
	if len(authorization) != 1 {
		return false
	}

	bearerToken := strings.TrimPrefix(authorization[0], "Bearer ")

	if token.IsSystemAdminToken(bearerToken) {
		_, err := token.ParseSystemAdminToken(systemAdminKey, bearerToken)
		return err == nil
	}

	return false
}
