package cmd

import (
	"context"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
)

func authMethodsCommand() *coral.Command {
	command := &coral.Command{
		Use:          "auth-methods",
		Short:        "Manage ionscale auth methods",
		SilenceUsage: true,
	}

	command.AddCommand(listAuthMethods())
	command.AddCommand(createAuthMethodCommand())

	return command
}

func listAuthMethods() *coral.Command {
	command := &coral.Command{
		Use:          "list",
		Short:        "List auth methods",
		SilenceUsage: true,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.RunE = func(command *coral.Command, args []string) error {

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		resp, err := client.ListAuthMethods(context.Background(), connect.NewRequest(&api.ListAuthMethodsRequest{}))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME", "TYPE")
		for _, m := range resp.Msg.AuthMethods {
			tbl.AddRow(m.Id, m.Name, m.Type)
		}
		tbl.Print()

		return nil
	}

	return command
}

func createAuthMethodCommand() *coral.Command {
	command := &coral.Command{
		Use:          "create",
		Short:        "Create a new auth method",
		SilenceUsage: true,
	}

	command.AddCommand(createOIDCAuthMethodCommand())

	return command
}

func createOIDCAuthMethodCommand() *coral.Command {
	command := &coral.Command{
		Use:          "oidc",
		Short:        "Create a new auth method",
		SilenceUsage: true,
	}

	var methodName string

	var clientId string
	var clientSecret string
	var issuer string

	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVarP(&methodName, "name", "n", "", "")
	command.Flags().StringVar(&clientId, "client-id", "", "")
	command.Flags().StringVar(&clientSecret, "client-secret", "", "")
	command.Flags().StringVar(&issuer, "issuer", "", "")

	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("client-id")
	_ = command.MarkFlagRequired("client-secret")
	_ = command.MarkFlagRequired("issuer")

	command.RunE = func(command *coral.Command, args []string) error {

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := &api.CreateAuthMethodRequest{
			Type:         "oidc",
			Name:         methodName,
			Issuer:       issuer,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		}

		resp, err := client.CreateAuthMethod(context.Background(), connect.NewRequest(req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME", "TYPE")
		tbl.AddRow(resp.Msg.AuthMethod.Id, resp.Msg.AuthMethod.Name, resp.Msg.AuthMethod.Type)
		tbl.Print()

		return nil
	}

	return command
}
