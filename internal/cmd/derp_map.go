package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"tailscale.com/tailcfg"
)

func systemCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "system",
		Short: "Manage global system configurations",
	}

	command.AddCommand(getDefaultDERPMap())

	return command
}

func getDefaultDERPMap() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "get-derp-map",
		Short:        "Get the default DERP Map configuration",
		SilenceUsage: true,
	})

	var asJson bool

	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		resp, err := tc.Client().GetDefaultDERPMap(cmd.Context(), connect.NewRequest(&api.GetDefaultDERPMapRequest{}))

		if err != nil {
			return err
		}

		var derpMap struct {
			Regions map[int]*tailcfg.DERPRegion
		}

		if err := json.Unmarshal(resp.Msg.Value, &derpMap); err != nil {
			return err
		}

		if asJson {
			marshal, err := json.MarshalIndent(derpMap, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(derpMap)
			if err != nil {
				return err
			}

			fmt.Println(string(marshal))
		}

		return nil
	}

	return command
}
