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
	"os"
	"strings"
	"text/tabwriter"
)

func machineCommands() *coral.Command {
	command := &coral.Command{
		Use:          "machines",
		Short:        "Manage ionscale machines",
		SilenceUsage: true,
	}

	command.AddCommand(getMachineCommand())
	command.AddCommand(deleteMachineCommand())
	command.AddCommand(expireMachineCommand())
	command.AddCommand(listMachinesCommand())
	command.AddCommand(getMachineRoutesCommand())
	command.AddCommand(enableMachineRoutesCommand())
	command.AddCommand(disableMachineRoutesCommand())
	command.AddCommand(enableMachineKeyExpiryCommand())
	command.AddCommand(disableMachineKeyExpiryCommand())

	return command
}

func getMachineCommand() *coral.Command {
	command := &coral.Command{
		Use:          "get",
		Short:        "Retrieve detailed information for a machine",
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

		req := api.GetMachineRequest{MachineId: machineID}
		resp, err := client.GetMachine(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		m := resp.Msg.Machine
		var lastSeen = "N/A"
		var expiresAt = "No expiry"

		if m.LastSeen != nil && !m.LastSeen.AsTime().IsZero() {
			if mom, err := goment.New(m.LastSeen.AsTime()); err == nil {
				lastSeen = mom.FromNow()
			}
		}

		if !m.KeyExpiryDisabled && m.ExpiresAt != nil && !m.ExpiresAt.AsTime().IsZero() {
			if mom, err := goment.New(m.ExpiresAt.AsTime()); !m.ExpiresAt.AsTime().IsZero() && err == nil {
				expiresAt = mom.FromNow()
			}
		}

		// initialize tabwriter
		w := new(tabwriter.Writer)

		// minwidth, tabwidth, padding, padchar, flags
		w.Init(os.Stdout, 8, 8, 0, '\t', 0)

		defer w.Flush()

		fmt.Fprintf(w, "%s\t%d\n", "ID", m.Id)
		fmt.Fprintf(w, "%s\t%s\n", "Machine name", m.Name)
		fmt.Fprintf(w, "%s\t%s\n", "Creator", m.User.Name)
		fmt.Fprintf(w, "%s\t%s\n", "OS", m.Os)
		fmt.Fprintf(w, "%s\t%s\n", "Tailscale version", m.ClientVersion)
		fmt.Fprintf(w, "%s\t%s\n", "Tailscale IPv4", m.Ipv4)
		fmt.Fprintf(w, "%s\t%s\n", "Tailscale IPv6", m.Ipv6)
		fmt.Fprintf(w, "%s\t%s\n", "Last seen", lastSeen)
		fmt.Fprintf(w, "%s\t%s\n", "Key expiry", expiresAt)

		for i, t := range m.Tags {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "ACL tags", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		for i, e := range m.ClientConnectivity.Endpoints {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Endpoints", e)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", e)
			}
		}

		for i, t := range m.AdvertisedRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Advertised routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		for i, t := range m.EnabledRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Enabled routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		return nil
	}

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

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 8, 8, 0, '\t', 0)
		defer w.Flush()

		for i, t := range resp.Msg.AdvertisedRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Advertised routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		for i, t := range resp.Msg.EnabledRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Enabled routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		return nil
	}

	return command
}

func enableMachineRoutesCommand() *coral.Command {
	command := &coral.Command{
		Use:          "enable-routes",
		Short:        "Enable routes for a given machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var routes []string
	var replace bool
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringSliceVar(&routes, "routes", []string{}, "List of routes to enable")
	command.Flags().BoolVar(&replace, "replace", false, "Replace current enabled routes with this new list")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		for _, r := range routes {
			if _, err := netaddr.ParseIPPrefix(r); err != nil {
				return err
			}
		}

		req := api.EnableMachineRoutesRequest{MachineId: machineID, Routes: routes, Replace: replace}
		resp, err := client.EnableMachineRoutes(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 8, 8, 0, '\t', 0)
		defer w.Flush()

		for i, t := range resp.Msg.AdvertisedRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Advertised routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		for i, t := range resp.Msg.EnabledRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Enabled routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		return nil
	}

	return command
}

func disableMachineRoutesCommand() *coral.Command {
	command := &coral.Command{
		Use:          "disable-routes",
		Short:        "Disable routes for a given machine",
		SilenceUsage: true,
	}

	var machineID uint64
	var routes []string
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringSliceVar(&routes, "routes", []string{}, "List of routes to enable")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(command *coral.Command, args []string) error {
		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		for _, r := range routes {
			if _, err := netaddr.ParseIPPrefix(r); err != nil {
				return err
			}
		}

		req := api.DisableMachineRoutesRequest{MachineId: machineID, Routes: routes}
		resp, err := client.DisableMachineRoutes(context.Background(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 8, 8, 0, '\t', 0)
		defer w.Flush()

		for i, t := range resp.Msg.AdvertisedRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Advertised routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

		for i, t := range resp.Msg.EnabledRoutes {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\n", "Enabled routes", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}

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
