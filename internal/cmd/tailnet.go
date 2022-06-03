package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
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
	command.AddCommand(deleteTailnetCommand())
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

		resp, err := client.ListTailnets(context.Background(), connect.NewRequest(&api.ListTailnetRequest{}))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME")
		for _, tailnet := range resp.Msg.Tailnet {
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

		resp, err := client.CreateTailnet(context.Background(), connect.NewRequest(&api.CreateTailnetRequest{Name: name}))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME")
		tbl.AddRow(resp.Msg.Tailnet.Id, resp.Msg.Tailnet.Name)
		tbl.Print()

		return nil
	}

	return command
}

func deleteTailnetCommand() *coral.Command {
	command := &coral.Command{
		Use:          "delete",
		Short:        "Delete a tailnet",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var force bool
	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")
	command.Flags().BoolVar(&force, "force", false, "")

	command.RunE = func(command *coral.Command, args []string) error {

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		_, err = client.DeleteTailnet(context.Background(), connect.NewRequest(&api.DeleteTailnetRequest{TailnetId: tailnet.Id, Force: force}))

		if err != nil {
			return err
		}

		fmt.Println("Tailnet deleted.")

		return nil
	}

	return command
}
