package cmd

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/muesli/coral"
)

const (
	ionscaleSystemAdminKey     = "IONSCALE_SYSTEM_ADMIN_KEY"
	ionscaleAddr               = "IONSCALE_ADDR"
	ionscaleInsecureSkipVerify = "IONSCALE_SKIP_VERIFY"
)

type Target struct {
	addr               string
	insecureSkipVerify bool
	systemAdminKey     string
}

func (t *Target) prepareCommand(cmd *coral.Command) {
	cmd.Flags().StringVar(&t.addr, "addr", "", "Addr of the ionscale server, as a complete URL")
	cmd.Flags().BoolVar(&t.insecureSkipVerify, "tls-skip-verify", false, "Disable verification of TLS certificates")
	cmd.Flags().StringVar(&t.systemAdminKey, "system-admin-key", "", "If specified, the given value will be used as the key to generate a Bearer token for the call. This can also be specified via the IONSCALE_ADMIN_KEY environment variable.")
}

func (t *Target) createGRPCClient() (api.IonscaleServiceClient, error) {
	addr := t.getAddr()
	skipVerify := t.getInsecureSkipVerify()
	systemAdminKey := t.getSystemAdminKey()

	auth, err := ionscale.LoadClientAuth(systemAdminKey)
	if err != nil {
		return nil, err
	}

	return ionscale.NewClient(auth, addr, skipVerify)
}

func (t *Target) getAddr() string {
	if len(t.addr) != 0 {
		return t.addr
	}
	return config.GetString(ionscaleAddr, "https://localhost:8443")
}

func (t *Target) getInsecureSkipVerify() bool {
	if t.insecureSkipVerify {
		return true
	}
	return config.GetBool(ionscaleInsecureSkipVerify, false)
}

func (t *Target) getSystemAdminKey() string {
	if len(t.systemAdminKey) != 0 {
		return t.systemAdminKey
	}
	return config.GetString(ionscaleSystemAdminKey, "")
}
