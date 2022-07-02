package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	apiconnect "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1/ionscalev1connect"
	"github.com/muesli/coral"
)

func checkAll(checks ...func(cmd *coral.Command, args []string) error) func(cmd *coral.Command, args []string) error {
	return func(cmd *coral.Command, args []string) error {
		for _, c := range checks {
			if err := c(cmd, args); err != nil {
				return err
			}
		}
		return nil
	}
}

func checkRequiredTailnetAndTailnetIdFlags(cmd *coral.Command, args []string) error {
	if !cmd.Flags().Changed("tailnet") && !cmd.Flags().Changed("tailnet-id") {
		return fmt.Errorf("flag --tailnet or --tailnet-id is required")
	}

	if cmd.Flags().Changed("tailnet") && cmd.Flags().Changed("tailnet-id") {
		return fmt.Errorf("flags --tailnet and --tailnet-id are mutually exclusive")
	}

	return nil
}

func findTailnet(client apiconnect.IonscaleServiceClient, tailnet string, tailnetID uint64) (*api.Tailnet, error) {
	if tailnetID == 0 && tailnet == "" {
		return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
	}

	tailnets, err := client.ListTailnets(context.Background(), connect.NewRequest(&api.ListTailnetRequest{}))
	if err != nil {
		return nil, err
	}

	for _, t := range tailnets.Msg.Tailnet {
		if t.Id == tailnetID || t.Name == tailnet {
			return t, nil
		}
	}

	return nil, fmt.Errorf("requested tailnet not found or you are not authorized for this tailnet")
}
