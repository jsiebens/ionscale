package server

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/service"
	"google.golang.org/grpc"
	"tailscale.com/types/key"
)

func init() {
	grpc_prometheus.EnableHandlingTimeHistogram()
}

func NewGrpcServer(logger hclog.Logger, systemAdminKey key.MachinePrivate) *grpc.Server {
	return grpc.NewServer(
		middleware.WithUnaryServerChain(
			logging.UnaryServerInterceptor(
				&grpcLogger{logger.Named("grpc")},
				logging.WithDurationField(logging.DurationToDurationField),
			),
			grpc_prometheus.UnaryServerInterceptor,
			recovery.UnaryServerInterceptor(),
			service.UnaryServerTokenAuth(systemAdminKey),
		),
	)
}

type grpcLogger struct {
	log hclog.Logger
}

func (l *grpcLogger) Log(lvl logging.Level, msg string) {
	switch lvl {
	case logging.ERROR:
		l.log.Error(msg)
	default:
		l.log.Debug(msg)
	}
}

func (l *grpcLogger) With(fields ...string) logging.Logger {
	if len(fields) == 0 {
		return l
	}
	vals := make([]interface{}, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		vals = append(vals, fields[i])
	}
	return &grpcLogger{log: l.log.With(vals...)}
}
