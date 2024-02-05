package cmd

import (
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
)

func authCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "auth",
	}

	command.AddCommand(authLoginCommand())

	return command
}

func authLoginCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "login",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := &api.AuthenticateRequest{}
		stream, err := tc.Client().Authenticate(cmd.Context(), connect.NewRequest(req))
		if err != nil {
			return err
		}

		var started = false
		for stream.Receive() {
			resp := stream.Msg()
			if len(resp.Token) != 0 {
				fmt.Println()
				fmt.Println("Success.")

				tailnetId := uint64(0)
				if resp.TailnetId != nil {
					tailnetId = *resp.TailnetId
				}

				if err := ionscale.StoreAuthToken(tc.Addr(), resp.Token, tailnetId); err != nil {
					fmt.Println()
					fmt.Println("Your api token:")
					fmt.Println()
					fmt.Printf("  %s\n", resp.Token)
					fmt.Println()
				}
				return nil
			}

			if len(resp.AuthUrl) != 0 && !started {
				started = true

				fmt.Println()
				fmt.Println("To authenticate, visit:")
				fmt.Println()
				fmt.Printf("  %s\n", resp.AuthUrl)
				fmt.Println()
			}
		}

		if err := stream.Err(); err != nil {
			return err
		}

		return nil
	}

	return command
}
