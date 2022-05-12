package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
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
	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")
	command.Flags().BoolVar(&asJson, "json", false, "")

	command.RunE = func(command *coral.Command, args []string) error {
		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		resp, err := client.GetACLPolicy(context.Background(), &api.GetACLPolicyRequest{TailnetId: tailnet.Id})
		if err != nil {
			return err
		}

		if asJson {
			marshal, err := json.MarshalIndent(resp.Policy, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(resp.Policy)
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
	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")
	command.Flags().StringVar(&file, "file", "", "")

	command.RunE = func(command *coral.Command, args []string) error {
		rawJson, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		var policy api.Policy

		if err := json.Unmarshal(rawJson, &policy); err != nil {
			return err
		}

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		_, err = client.SetACLPolicy(context.Background(), &api.SetACLPolicyRequest{TailnetId: tailnet.Id, Policy: &policy})
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("ACL policy updated successfully")

		return nil
	}

	return command
}
