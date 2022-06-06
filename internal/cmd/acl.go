package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func getACLConfig() *coral.Command {
	command := &coral.Command{
		Use:          "get-acl",
		Short:        "Get the ACL policy",
		SilenceUsage: true,
	}

	var asJson bool
	var tailnetID uint64
	var tailnetName string
	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

	command.PreRunE = checkRequiredTailnetAndTailnetIdFlags
	command.RunE = func(cmd *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		resp, err := client.GetACLPolicy(context.Background(), connect.NewRequest(&api.GetACLPolicyRequest{TailnetId: tailnet.Id}))
		if err != nil {
			return err
		}

		var p domain.ACLPolicy

		if err := json.Unmarshal(resp.Msg.Value, &p); err != nil {
			return err
		}

		if asJson {
			marshal, err := json.MarshalIndent(&p, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(&p)
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

func setACLConfig() *coral.Command {
	command := &coral.Command{
		Use:          "set-acl",
		Short:        "Set ACL policy",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var file string
	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().StringVar(&file, "file", "", "Path to json file with the acl configuration")

	command.PreRunE = checkRequiredTailnetAndTailnetIdFlags
	command.RunE = func(cmd *coral.Command, args []string) error {
		rawJson, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		_, err = client.SetACLPolicy(context.Background(), connect.NewRequest(&api.SetACLPolicyRequest{TailnetId: tailnet.Id, Value: rawJson}))
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("ACL policy updated successfully")

		return nil
	}

	return command
}
