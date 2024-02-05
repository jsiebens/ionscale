package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/go-edit/editor"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
	"github.com/tailscale/hujson"
	"os"
)

func getACLConfigCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "get-acl-policy",
		Short:        "Get the ACL policy",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		resp, err := tc.Client().GetACLPolicy(cmd.Context(), connect.NewRequest(&api.GetACLPolicyRequest{TailnetId: tc.TailnetID()}))
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

func editACLConfigCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "edit-acl-policy",
		Short:        "Edit the ACL policy",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		edit := editor.NewDefaultEditor([]string{"IONSCALE_EDITOR", "EDITOR"})

		resp, err := tc.Client().GetACLPolicy(cmd.Context(), connect.NewRequest(&api.GetACLPolicyRequest{TailnetId: tc.TailnetID()}))
		if err != nil {
			return err
		}

		previous, err := json.MarshalIndent(resp.Msg.Policy, "", "  ")
		if err != nil {
			return err
		}

		next, s, err := edit.LaunchTempFile("ionscale", ".json", bytes.NewReader(previous))
		if err != nil {
			return err
		}

		defer os.Remove(s)

		next, err = hujson.Standardize(next)
		if err != nil {
			return err
		}

		var policy = &api.ACLPolicy{}
		if err := json.Unmarshal(next, policy); err != nil {
			return err
		}

		_, err = tc.Client().SetACLPolicy(cmd.Context(), connect.NewRequest(&api.SetACLPolicyRequest{TailnetId: tc.TailnetID(), Policy: policy}))
		if err != nil {
			return err
		}

		fmt.Println("ACL policy updated successfully")

		return nil
	}

	return command
}

func setACLConfigCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "set-acl-policy",
		Short:        "Set ACL policy",
		SilenceUsage: true,
	})

	var file string

	command.Flags().StringVar(&file, "file", "", "Path to json file with the acl configuration")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		rawJson, err := hujson.Standardize(content)
		if err != nil {
			return err
		}

		var policy = &api.ACLPolicy{}
		if err := json.Unmarshal(rawJson, policy); err != nil {
			return err
		}

		_, err = tc.Client().SetACLPolicy(cmd.Context(), connect.NewRequest(&api.SetACLPolicyRequest{TailnetId: tc.TailnetID(), Policy: policy}))
		if err != nil {
			return err
		}

		fmt.Println("ACL policy updated successfully")

		return nil
	}

	return command
}
