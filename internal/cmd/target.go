package cmd

import (
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	ionscalev1 "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/spf13/cobra"
)

const (
	ionscaleSystemAdminKey     = "IONSCALE_SYSTEM_ADMIN_KEY"
	ionscaleKeysSystemAdminKey = "IONSCALE_KEYS_SYSTEM_ADMIN_KEY"
	ionscaleAddr               = "IONSCALE_ADDR"
	ionscaleInsecureSkipVerify = "IONSCALE_SKIP_VERIFY"
)

type TargetContext interface {
	Client() api.IonscaleServiceClient
	Addr() string
	TailnetID() uint64
}

type target struct {
	addr               string
	insecureSkipVerify bool
	systemAdminKey     string

	tailnetID   uint64
	tailnetName string

	client  api.IonscaleServiceClient
	tailnet *ionscalev1.Tailnet
}

func prepareCommand(enableTailnetSelector bool, cmd *cobra.Command) (*cobra.Command, TargetContext) {
	t := &target{}

	cmd.Flags().StringVar(&t.addr, "addr", "", "Addr of the ionscale server, as a complete URL")
	cmd.Flags().BoolVar(&t.insecureSkipVerify, "tls-skip-verify", false, "Disable verification of TLS certificates")
	cmd.Flags().StringVar(&t.systemAdminKey, "system-admin-key", "", "If specified, the given value will be used as the key to generate a Bearer token for the call. This can also be specified via the IONSCALE_ADMIN_KEY environment variable.")

	if enableTailnetSelector {
		cmd.Flags().StringVar(&t.tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
		cmd.Flags().Uint64Var(&t.tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	}

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		addr := t.getAddr()
		skipVerify := t.getInsecureSkipVerify()
		systemAdminKey := t.getSystemAdminKey()

		auth, err := ionscale.LoadClientAuth(systemAdminKey)
		if err != nil {
			return err
		}

		client, err := ionscale.NewClient(auth, addr, skipVerify)
		if err != nil {
			return err
		}

		t.client = client

		if enableTailnetSelector {
			savedTailnetID, err := ionscale.TailnetFromFile()
			if err != nil {
				return err
			}

			if savedTailnetID == 0 && !cmd.Flags().Changed("tailnet") && !cmd.Flags().Changed("tailnet-id") {
				return fmt.Errorf("flag --tailnet or --tailnet-id is required")
			}

			if cmd.Flags().Changed("tailnet") && cmd.Flags().Changed("tailnet-id") {
				return fmt.Errorf("flags --tailnet and --tailnet-id are mutually exclusive")
			}

			tailnets, err := t.client.ListTailnets(cmd.Context(), connect.NewRequest(&ionscalev1.ListTailnetsRequest{}))
			if err != nil {
				return err
			}

			for _, tailnet := range tailnets.Msg.Tailnet {
				if tailnet.Id == savedTailnetID || tailnet.Id == t.tailnetID || tailnet.Name == t.tailnetName {
					t.tailnet = tailnet
					break
				}
			}

			if t.tailnet == nil {
				return fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
			}
		}

		return nil
	}

	return cmd, t
}

func (t *target) getAddr() string {
	if len(t.addr) != 0 {
		return t.addr
	}
	return config.GetString(ionscaleAddr, "https://localhost:8443")
}

func (t *target) getInsecureSkipVerify() bool {
	if t.insecureSkipVerify {
		return true
	}
	return config.GetBool(ionscaleInsecureSkipVerify, false)
}

func (t *target) getSystemAdminKey() string {
	if len(t.systemAdminKey) != 0 {
		return t.systemAdminKey
	}
	return config.GetString(ionscaleSystemAdminKey, config.GetString(ionscaleKeysSystemAdminKey, ""))
}

func (t *target) Addr() string {
	return t.getAddr()
}

func (t *target) Client() api.IonscaleServiceClient {
	return t.client
}

func (t *target) TailnetID() uint64 {
	if t.tailnet == nil {
		return 0
	}
	return t.tailnet.Id
}
