package cmd

import (
	"bytes"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/go-edit/editor"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
	"os"
)

func getIAMPolicyCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "get-iam-policy",
		Short:        "Get the IAM policy",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		resp, err := tc.Client().GetIAMPolicy(cmd.Context(), connect.NewRequest(&api.GetIAMPolicyRequest{TailnetId: tc.TailnetID()}))
		if err != nil {
			return err
		}

		fmt.Println(resp.Msg.Policy)

		return nil
	}

	return command
}

func editIAMPolicyCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "edit-iam-policy",
		Short:        "Edit the IAM policy",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		edit := editor.NewDefaultEditor([]string{"IONSCALE_EDITOR", "EDITOR"})

		resp, err := tc.Client().GetIAMPolicy(cmd.Context(), connect.NewRequest(&api.GetIAMPolicyRequest{TailnetId: tc.TailnetID()}))
		if err != nil {
			return err
		}

		next, s, err := edit.LaunchTempFile("ionscale", ".json", bytes.NewReader([]byte(resp.Msg.Policy)))
		if err != nil {
			return err
		}

		defer os.Remove(s)

		_, err = tc.Client().SetIAMPolicy(cmd.Context(), connect.NewRequest(&api.SetIAMPolicyRequest{TailnetId: tc.TailnetID(), Policy: string(next)}))
		if err != nil {
			return err
		}

		fmt.Println("IAM policy updated successfully")

		return nil
	}

	return command
}

func setIAMPolicyCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "set-iam-policy",
		Short:        "Set IAM policy",
		SilenceUsage: true,
	})

	var file string

	command.Flags().StringVar(&file, "file", "", "Path to json file with the acl configuration")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		_, err = tc.Client().SetIAMPolicy(cmd.Context(), connect.NewRequest(&api.SetIAMPolicyRequest{TailnetId: tc.TailnetID(), Policy: string(content)}))
		if err != nil {
			return err
		}

		fmt.Println("IAM policy updated successfully")

		return nil
	}

	return command
}
