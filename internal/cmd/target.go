package cmd

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"github.com/muesli/coral"
	"io"
)

const (
	ionscaleSystemAdminKey     = "IONSCALE_ADMIN_KEY"
	ionscaleAddr               = "IONSCALE_ADDR"
	ionscaleInsecureSkipVerify = "IONSCALE_SKIP_VERIFY"
	ionscaleUseGrpcWeb         = "IONSCALE_GRPC_WEB"
)

type Target struct {
	addr               string
	useGrpcWeb         bool
	insecureSkipVerify bool
	systemAdminKey     string
}

func (t *Target) prepareCommand(cmd *coral.Command) {
	cmd.Flags().StringVar(&t.addr, "addr", "", "Addr of the ionscale server, as a complete URL")
	cmd.Flags().BoolVar(&t.insecureSkipVerify, "tls-skip-verify", false, "Disable verification of TLS certificates")
	cmd.Flags().BoolVar(&t.useGrpcWeb, "grpc-web", false, "Enables gRPC-web protocol. Useful if ionscale server is behind proxy which does not support GRPC")
	cmd.Flags().StringVar(&t.systemAdminKey, "admin-key", "", "If specified, the given value will be used as the key to generate a Bearer token for the call. This can also be specified via the IONSCALE_ADMIN_KEY environment variable.")
}

func (t *Target) createGRPCClient() (api.IonscaleClient, io.Closer, error) {
	addr := t.getAddr()
	useGrpcWeb := t.getUseGrpcWeb()
	skipVerify := t.getInsecureSkipVerify()
	systemAdminKey := t.getSystemAdminKey()

	auth, err := ionscale.LoadClientAuth(systemAdminKey)
	if err != nil {
		return nil, nil, err
	}

	return ionscale.NewClient(auth, addr, skipVerify, useGrpcWeb)
}

func (t *Target) getAddr() string {
	if len(t.addr) != 0 {
		return t.addr
	}
	return config.GetString(ionscaleAddr, "https://localhost:8000")
}

func (t *Target) getInsecureSkipVerify() bool {
	if t.insecureSkipVerify {
		return true
	}
	return config.GetBool(ionscaleInsecureSkipVerify, false)
}

func (t *Target) getUseGrpcWeb() bool {
	if t.useGrpcWeb {
		return true
	}
	return config.GetBool(ionscaleUseGrpcWeb, false)
}

func (t *Target) getSystemAdminKey() string {
	if len(t.systemAdminKey) != 0 {
		return t.systemAdminKey
	}
	return config.GetString(ionscaleSystemAdminKey, "")
}
