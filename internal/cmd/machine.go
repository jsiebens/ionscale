package cmd

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"github.com/muesli/coral"
	"github.com/nleeper/goment"
	"github.com/rodaine/table"
)

func machineCommands() *coral.Command {
	command := &coral.Command{
		Use:          "machines",
		Short:        "Manage ionscale machines",
		SilenceUsage: true,
	}

	command.AddCommand(deleteMachineCommand())
	command.AddCommand(listMachinesCommand())

	return command
}

func deleteMachineCommand() *coral.Command {
	command := &coral.Command{
		Use:          "delete",
		Short:        "Deletes a machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "")

	command.RunE = func(command *coral.Command, args []string) error {
		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		req := api.DeleteMachineRequest{MachineId: machineID}
		if _, err := client.DeleteMachine(context.Background(), &req); err != nil {
			return err
		}

		fmt.Println("Machine deleted.")

		return nil
	}

	return command
}

func listMachinesCommand() *coral.Command {
	command := &coral.Command{
		Use:          "list",
		Short:        "List machines",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string

	var target = Target{}
	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")

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

		req := api.ListMachinesRequest{TailnetId: tailnet.Id}
		resp, err := client.ListMachines(context.Background(), &req)

		if err != nil {
			return err
		}

		tbl := table.New("ID", "TAILNET", "NAME", "IPv4", "IPv6", "EPHEMERAL", "LAST_SEEN", "USER")
		for _, m := range resp.Machines {
			var lastSeen = "N/A"
			if m.Connected {
				lastSeen = "Connected"
			} else if m.LastSeen != nil {
				mom, err := goment.New(m.LastSeen.AsTime())
				if err == nil {
					lastSeen = mom.FromNow()
				}
			}
			tbl.AddRow(m.Id, m.Tailnet.Name, m.Name, m.Ipv4, m.Ipv6, m.Ephemeral, lastSeen, m.User.Name)
		}
		tbl.Print()

		return nil
	}

	return command
}
