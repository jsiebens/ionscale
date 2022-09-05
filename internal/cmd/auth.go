package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
)

func authCommand() *coral.Command {
	command := &coral.Command{
		Use: "auth",
	}

	command.AddCommand(authLoginCommand())

	return command
}

func authLoginCommand() *coral.Command {
	command := &coral.Command{
		Use:          "login",
		SilenceUsage: true,
	}

	var target = Target{}

	target.prepareCommand(command)

	command.RunE = func(command *coral.Command, args []string) error {

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := &api.AuthenticationRequest{}
		stream, err := client.Authenticate(context.Background(), connect.NewRequest(req))
		if err != nil {
			return err
		}

		var started = false
		for stream.Receive() {
			resp := stream.Msg()
			if len(resp.Token) != 0 {
				fmt.Println()
				fmt.Println("Success.")
				if err := ionscale.SessionToFile(resp.Token, resp.TailnetId); err != nil {
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
