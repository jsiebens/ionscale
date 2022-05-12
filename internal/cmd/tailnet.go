package cmd

import (
	"context"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
)

func tailnetCommand() *coral.Command {
	command := &coral.Command{
		Use:   "tailnets",
		Short: "Manage ionscale tailnets",
		Long:  "This command allows operations on ionscale tailnet resources.",
	}

	command.AddCommand(listTailnetsCommand())
	command.AddCommand(createTailnetsCommand())
	command.AddCommand(getDNSConfig())
	command.AddCommand(setDNSConfig())
	command.AddCommand(getACLConfig())
	command.AddCommand(setACLConfig())

	return command
}

func listTailnetsCommand() *coral.Command {
	command := &coral.Command{
		Use:          "list",
		Short:        "List tailnets",
		Long:         `List tailnets in this ionscale instance.`,
		SilenceUsage: true,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.RunE = func(command *coral.Command, args []string) error {

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		resp, err := client.ListTailnets(context.Background(), &api.ListTailnetRequest{})

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME")
		for _, tailnet := range resp.Tailnet {
			tbl.AddRow(tailnet.Id, tailnet.Name)
		}
		tbl.Print()

		return nil
	}

	return command
}

func createTailnetsCommand() *coral.Command {
	command := &coral.Command{
		Use:          "create",
		Short:        "Create a new tailnet",
		Long:         `List tailnets in this ionscale instance.`,
		SilenceUsage: true,
	}

	var name string
	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVarP(&name, "name", "n", "", "")
	_ = command.MarkFlagRequired("name")

	command.RunE = func(command *coral.Command, args []string) error {

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		resp, err := client.CreateTailnet(context.Background(), &api.CreateTailnetRequest{Name: name})

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME")
		tbl.AddRow(resp.Tailnet.Id, resp.Tailnet.Name)
		tbl.Print()

		return nil
	}

	return command
}
