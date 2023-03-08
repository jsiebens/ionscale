package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

func userCommands() *cobra.Command {
	command := &cobra.Command{
		Use:          "users",
		Aliases:      []string{"user"},
		Short:        "Manage ionscale users",
		SilenceUsage: true,
	}

	command.AddCommand(listUsersCommand())
	command.AddCommand(deleteUserCommand())

	return command
}

func listUsersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "list",
		Short:        "List users",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string

	var target = Target{}
	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")

	command.PreRunE = checkRequiredTailnetAndTailnetIdFlags
	command.RunE = func(command *cobra.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		req := api.ListUsersRequest{TailnetId: tailnet.Id}
		resp, err := client.ListUsers(context.Background(), connect.NewRequest(&req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "USER", "ROLE")
		for _, m := range resp.Msg.Users {
			tbl.AddRow(m.Id, m.Name, m.Role)
		}
		tbl.Print()

		return nil
	}

	return command
}

func deleteUserCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "delete",
		Short:        "Deletes a user",
		SilenceUsage: true,
	}

	var userID uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&userID, "user-id", 0, "User ID.")

	_ = command.MarkFlagRequired("user-id")

	command.RunE = func(command *cobra.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := api.DeleteUserRequest{UserId: userID}
		if _, err := client.DeleteUser(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("User deleted.")

		return nil
	}

	return command
}
