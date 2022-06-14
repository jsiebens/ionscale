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

func getIAMPolicyCommand() *coral.Command {
	command := &coral.Command{
		Use:          "get-iam-policy",
		Short:        "Get the IAM policy",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")

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

		resp, err := client.GetIAMPolicy(context.Background(), connect.NewRequest(&api.GetIAMPolicyRequest{TailnetId: tailnet.Id}))
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

func setIAMPolicyCommand() *coral.Command {
	command := &coral.Command{
		Use:          "set-iam-policy",
		Short:        "Set IAM policy",
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

		var policy = &api.IAMPolicy{}
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

		_, err = client.SetIAMPolicy(context.Background(), connect.NewRequest(&api.SetIAMPolicyRequest{TailnetId: tailnet.Id, Policy: policy}))
		if err != nil {
			return err
		}

		fmt.Println("IAM policy updated successfully")

		return nil
	}

	return command
}
