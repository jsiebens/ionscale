package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/hashicorp/go-bexpr"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
)

func authFilterCommand() *coral.Command {
	command := &coral.Command{
		Use:   "auth-filters",
		Short: "Manage ionscale auth filters",
		Long: `This command allows operations on ionscale auth filter resources. Example:

      $ ionscale auth-filter create`,
	}

	command.AddCommand(createAuthFilterCommand())
	command.AddCommand(listAuthFilterCommand())
	command.AddCommand(deleteAuthFilterCommand())

	return command
}

func listAuthFilterCommand() *coral.Command {
	command := &coral.Command{
		Use:          "list",
		SilenceUsage: true,
	}

	var authMethodID uint64

	var target = Target{}
	target.prepareCommand(command)

	command.Flags().Uint64Var(&authMethodID, "auth-method-id", 0, "")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := &api.ListAuthFiltersRequest{}

		if authMethodID != 0 {
			req.AuthMethodId = &authMethodID
		}

		resp, err := client.ListAuthFilters(context.Background(), connect.NewRequest(req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "AUTH_METHOD", "TAILNET", "EXPR")
		for _, filter := range resp.Msg.AuthFilters {
			if filter.Tailnet != nil {
				tbl.AddRow(filter.Id, filter.AuthMethod.Name, filter.Tailnet.Name, filter.Expr)
			} else {
				tbl.AddRow(filter.Id, filter.AuthMethod.Name, "", filter.Expr)
			}
		}
		tbl.Print()

		return nil
	}

	return command
}

func createAuthFilterCommand() *coral.Command {
	command := &coral.Command{
		Use:          "create",
		SilenceUsage: true,
	}

	var expr string
	var tailnetID uint64
	var tailnetName string
	var authMethodID uint64
	var authMethodName string

	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVar(&expr, "expr", "*", "")
	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")
	command.Flags().StringVar(&authMethodName, "auth-method", "", "")
	command.Flags().Uint64Var(&authMethodID, "auth-method-id", 0, "")

	command.RunE = func(command *coral.Command, args []string) error {
		if expr != "*" {
			if _, err := bexpr.CreateEvaluator(expr); err != nil {
				return fmt.Errorf("invalid expression: %v", err)
			}
		}

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		authMethod, err := findAuthMethod(client, authMethodName, authMethodID)
		if err != nil {
			return err
		}

		req := &api.CreateAuthFilterRequest{
			AuthMethodId: authMethod.Id,
			TailnetId:    tailnet.Id,
			Expr:         expr,
		}

		resp, err := client.CreateAuthFilter(context.Background(), connect.NewRequest(req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "AUTH_METHOD", "TAILNET", "EXPR")
		if resp.Msg.AuthFilter.Tailnet != nil {
			tbl.AddRow(resp.Msg.AuthFilter.Id, resp.Msg.AuthFilter.AuthMethod.Name, resp.Msg.AuthFilter.Tailnet.Name, resp.Msg.AuthFilter.Expr)
		} else {
			tbl.AddRow(resp.Msg.AuthFilter.Id, resp.Msg.AuthFilter.AuthMethod.Name, "", resp.Msg.AuthFilter.Expr)
		}
		tbl.Print()

		return nil
	}

	return command
}

func deleteAuthFilterCommand() *coral.Command {
	command := &coral.Command{
		Use:          "delete",
		SilenceUsage: true,
	}

	var authFilterID uint64

	var target = Target{}
	target.prepareCommand(command)

	command.Flags().Uint64Var(&authFilterID, "auth-filter-id", 0, "")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := &api.DeleteAuthFilterRequest{
			AuthFilterId: authFilterID,
		}

		_, err = client.DeleteAuthFilter(context.Background(), connect.NewRequest(req))

		if err != nil {
			return err
		}

		return nil
	}

	return command
}
