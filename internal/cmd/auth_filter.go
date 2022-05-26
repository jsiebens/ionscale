package cmd

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-bexpr"
	"github.com/jsiebens/ionscale/pkg/gen/api"
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
		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		req := &api.ListAuthFiltersRequest{}

		if authMethodID != 0 {
			req.AuthMethodId = &authMethodID
		}

		resp, err := client.ListAuthFilters(context.Background(), req)

		if err != nil {
			return err
		}

		tbl := table.New("ID", "AUTH_METHOD", "TAILNET", "EXPR")
		for _, filter := range resp.AuthFilters {
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

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

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

		resp, err := client.CreateAuthFilter(context.Background(), req)

		if err != nil {
			return err
		}

		tbl := table.New("ID", "AUTH_METHOD", "TAILNET", "EXPR")
		if resp.AuthFilter.Tailnet != nil {
			tbl.AddRow(resp.AuthFilter.Id, resp.AuthFilter.AuthMethod.Name, resp.AuthFilter.Tailnet.Name, resp.AuthFilter.Expr)
		} else {
			tbl.AddRow(resp.AuthFilter.Id, resp.AuthFilter.AuthMethod.Name, "", resp.AuthFilter.Expr)
		}
		tbl.Print()

		return nil
	}

	return command
}
