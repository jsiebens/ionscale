package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"io/ioutil"
)

func getACLConfigCommand() *coral.Command {
	command := &coral.Command{
		Use:          "get-acl-policy",
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

		marshal, err := json.MarshalIndent(resp.Msg.Policy, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(marshal))

		return nil
	}

	return command
}

func setACLConfigCommand() *coral.Command {
	command := &coral.Command{
		Use:          "set-acl-policy",
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

		var policy = &api.ACLPolicy{}
		if err := json.Unmarshal(rawJson, policy); err != nil {
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

		_, err = client.SetACLPolicy(context.Background(), connect.NewRequest(&api.SetACLPolicyRequest{TailnetId: tailnet.Id, Policy: policy}))
		if err != nil {
			return err
		}

		fmt.Println("ACL policy updated successfully")

		return nil
	}

	return command
}
