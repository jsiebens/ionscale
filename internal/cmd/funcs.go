package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	apiconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/spf13/cobra"
)

func checkRequiredTailnetAndTailnetIdFlags(cmd *cobra.Command, args []string) error {
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

	return nil
}

func findTailnet(client apiconnect.IonscaleServiceClient, tailnet string, tailnetID uint64) (*api.Tailnet, error) {
	savedTailnetID, err := ionscale.TailnetFromFile()
	if err != nil {
		return nil, err
	}

	if savedTailnetID == 0 && tailnetID == 0 && tailnet == "" {
		return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
	}

	tailnets, err := client.ListTailnets(context.Background(), connect.NewRequest(&api.ListTailnetsRequest{}))
	if err != nil {
		return nil, err
	}

	for _, t := range tailnets.Msg.Tailnet {
		if t.Id == savedTailnetID || t.Id == tailnetID || t.Name == tailnet {
			return t, nil
		}
	}

	return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
}
