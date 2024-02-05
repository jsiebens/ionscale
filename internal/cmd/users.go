package cmd

import (
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
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "list",
		Short:        "List users",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.ListUsersRequest{TailnetId: tc.TailnetID()}
		resp, err := tc.Client().ListUsers(cmd.Context(), connect.NewRequest(&req))

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
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "delete",
		Short:        "Deletes a user",
		SilenceUsage: true,
	})

	var userID uint64

	command.Flags().Uint64Var(&userID, "user-id", 0, "User ID.")

	_ = command.MarkFlagRequired("user-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DeleteUserRequest{UserId: userID}
		if _, err := tc.Client().DeleteUser(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("User deleted.")

		return nil
	}

	return command
}
