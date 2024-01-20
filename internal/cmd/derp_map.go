package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
	"github.com/tailscale/hujson"
	"gopkg.in/yaml.v2"
	"os"
	"tailscale.com/tailcfg"
)

func systemCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "system",
		Short: "Manage global system configurations",
	}

	command.AddCommand(getDefaultDERPMap())
	command.AddCommand(setDefaultDERPMap())
	command.AddCommand(resetDefaultDERPMap())

	return command
}

func getDefaultDERPMap() *cobra.Command {
	command := &cobra.Command{
		Use:          "get-derp-map",
		Short:        "Get the DERP Map configuration",
		SilenceUsage: true,
	}

	var asJson bool

	var target = Target{}
	target.prepareCommand(command)
	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

	command.RunE = func(command *cobra.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		resp, err := client.GetDefaultDERPMap(context.Background(), connect.NewRequest(&api.GetDefaultDERPMapRequest{}))

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

func setDefaultDERPMap() *cobra.Command {
	command := &cobra.Command{
		Use:          "set-derp-map",
		Short:        "Set the DERP Map configuration",
		SilenceUsage: true,
	}

	var file string
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().StringVar(&file, "file", "", "Path to json file with the DERP Map configuration")

	command.RunE = func(command *cobra.Command, args []string) error {
		grpcClient, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		rawJson, err := hujson.Standardize(content)
		if err != nil {
			return err
		}

		resp, err := grpcClient.SetDefaultDERPMap(context.Background(), connect.NewRequest(&api.SetDefaultDERPMapRequest{Value: rawJson}))
		if err != nil {
			return err
		}

		var derpMap tailcfg.DERPMap
		if err := json.Unmarshal(resp.Msg.Value, &derpMap); err != nil {
			return err
		}

		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}

func resetDefaultDERPMap() *cobra.Command {
	command := &cobra.Command{
		Use:          "reset-derp-map",
		Short:        "Reset the DERP Map to the default configuration",
		SilenceUsage: true,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.RunE = func(command *cobra.Command, args []string) error {
		grpcClient, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		if _, err := grpcClient.ResetDefaultDERPMap(context.Background(), connect.NewRequest(&api.ResetDefaultDERPMapRequest{})); err != nil {
			return err
		}

		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}
