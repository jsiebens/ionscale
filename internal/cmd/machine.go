package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"github.com/nleeper/goment"
	"github.com/rodaine/table"
	"inet.af/netaddr"
	"strings"
)

func machineCommands() *coral.Command {
	command := &coral.Command{
		Use:          "machines",
		Short:        "Manage ionscale machines",
		SilenceUsage: true,
	}

	command.AddCommand(deleteMachineCommand())
	command.AddCommand(expireMachineCommand())
	command.AddCommand(listMachinesCommand())
	command.AddCommand(getMachineRoutesCommand())
	command.AddCommand(setMachineRoutesCommand())
	command.AddCommand(enableMachineKeyExpiryCommand())
	command.AddCommand(disableMachineKeyExpiryCommand())

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
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := api.DeleteMachineRequest{MachineId: machineID}
		if _, err := client.DeleteMachine(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine deleted.")

		return nil
	}

	return command
}

func expireMachineCommand() *coral.Command {
	command := &coral.Command{
		Use:          "expire",
		Short:        "Expires a machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := api.ExpireMachineRequest{MachineId: machineID}
		if _, err := client.ExpireMachine(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine key expired.")

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
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")

	command.PreRunE = checkRequiredTailnetAndTailnetIdFlags
	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		tailnet, err := findTailnet(client, tailnetName, tailnetID)
		if err != nil {
			return err
		}

		req := api.ListMachinesRequest{TailnetId: tailnet.Id}
		resp, err := client.ListMachines(context.Background(), connect.NewRequest(&req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "TAILNET", "NAME", "IPv4", "IPv6", "EPHEMERAL", "LAST_SEEN", "TAGS")
		for _, m := range resp.Msg.Machines {
			var lastSeen = "N/A"
			if m.Connected {
				lastSeen = "Connected"
			} else if m.LastSeen != nil {
				mom, err := goment.New(m.LastSeen.AsTime())
				if err == nil {
					lastSeen = mom.FromNow()
				}
			}
			tbl.AddRow(m.Id, m.Tailnet.Name, m.Name, m.Ipv4, m.Ipv6, m.Ephemeral, lastSeen, strings.Join(m.Tags, ","))
		}
		tbl.Print()

		return nil
	}

	return command
}

func getMachineRoutesCommand() *coral.Command {
	command := &coral.Command{
		Use:          "get-routes",
		Short:        "Show routes advertised and enabled by a given machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		grpcClient, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := api.GetMachineRoutesRequest{MachineId: machineID}
		resp, err := grpcClient.GetMachineRoutes(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		tbl := table.New("ROUTE", "ALLOWED")
		for _, r := range resp.Msg.Routes {
			tbl.AddRow(r.Advertised, r.Allowed)
		}
		tbl.Print()

		return nil
	}

	return command
}

func setMachineRoutesCommand() *coral.Command {
	command := &coral.Command{
		Use:          "set-routes",
		Short:        "Set the enabled routes for a given machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var allowedIps []string
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringSliceVar(&allowedIps, "allowed-ips", []string{}, "List of routes to enable")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		var prefixes []netaddr.IPPrefix
		for _, r := range allowedIps {
			p, err := netaddr.ParseIPPrefix(r)
			if err != nil {
				return err
			}
			prefixes = append(prefixes, p)
		}

		req := api.SetMachineRoutesRequest{MachineId: machineID, AllowedIps: allowedIps}
		resp, err := client.SetMachineRoutes(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		tbl := table.New("ROUTE", "ALLOWED")
		for _, r := range resp.Msg.Routes {
			tbl.AddRow(r.Advertised, r.Allowed)
		}
		tbl.Print()

		return nil
	}

	return command
}

func enableMachineKeyExpiryCommand() *coral.Command {
	command := &coral.Command{
		Use:          "enable-key-expiry",
		Short:        "Enable machine key expiry",
		SilenceUsage: true,
	}

	return configureSetMachineKeyExpiryCommand(command, false)
}

func disableMachineKeyExpiryCommand() *coral.Command {
	command := &coral.Command{
		Use:          "disable-key-expiry",
		Short:        "Disable machine key expiry",
		SilenceUsage: true,
	}

	return configureSetMachineKeyExpiryCommand(command, true)
}

func configureSetMachineKeyExpiryCommand(command *coral.Command, v bool) *coral.Command {
	var machineID uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		req := api.SetMachineKeyExpiryRequest{MachineId: machineID, Disabled: v}
		_, err = client.SetMachineKeyExpiry(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		return nil
	}

	return command
}
