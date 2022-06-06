package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"tailscale.com/tailcfg"
)

func derpMapCommand() *coral.Command {
	command := &coral.Command{
		Use:   "derp-map",
		Short: "Manage DERP Map configuration",
	}

	command.AddCommand(getDERPMap())
	command.AddCommand(setDERPMap())

	return command
}

func getDERPMap() *coral.Command {
	command := &coral.Command{
		Use:          "get",
		Short:        "Get the DERP Map configuration",
		SilenceUsage: true,
	}

	var asJson bool

	var target = Target{}
	target.prepareCommand(command)
	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		resp, err := client.GetDERPMap(context.Background(), connect.NewRequest(&api.GetDERPMapRequest{}))

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

			fmt.Println()
			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(derpMap)
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println(string(marshal))
		}

		return nil
	}

	return command
}

func setDERPMap() *coral.Command {
	command := &coral.Command{
		Use:          "set",
		Short:        "Set the DERP Map configuration",
		SilenceUsage: true,
	}

	var file string
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().StringVar(&file, "file", "", "Path to json file with the DERP Map configuration")

	command.RunE = func(command *coral.Command, args []string) error {
		grpcClient, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		rawJson, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		resp, err := grpcClient.SetDERPMap(context.Background(), connect.NewRequest(&api.SetDERPMapRequest{Value: rawJson}))
		if err != nil {
			return err
		}

		var derpMap tailcfg.DERPMap
		if err := json.Unmarshal(resp.Msg.Value, &derpMap); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}
