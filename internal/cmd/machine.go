package cmd

import (
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/nleeper/goment"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"inet.af/netaddr"
	"os"
	"strings"
	"text/tabwriter"
)

func machineCommands() *cobra.Command {
	command := &cobra.Command{
		Use:          "machines",
		Aliases:      []string{"machine", "devices", "device"},
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
	command.AddCommand(enableExitNodeCommand())
	command.AddCommand(disableExitNodeCommand())
	command.AddCommand(disableMachineKeyExpiryCommand())
	command.AddCommand(authorizeMachineCommand())
	command.AddCommand(renameMachineCommand())

	return command
}

func getMachineCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "get",
		Short:        "Retrieve detailed information for a machine",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.GetMachineRequest{MachineId: machineID}
		resp, err := tc.Client().GetMachine(cmd.Context(), connect.NewRequest(&req))
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
		fmt.Fprintf(w, "%s\t%v\n", "Ephemeral", m.Ephemeral)
		if !m.Authorized {
			fmt.Fprintf(w, "%s\t%v\n", "Authorized", m.Authorized)
		}
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

		if m.AdvertisedExitNode {
			if m.EnabledExitNode {
				fmt.Fprintf(w, "%s\t%s\n", "Exit node", "enabled")
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "Exit node", "disabled")
			}
		} else {
			fmt.Fprintf(w, "%s\t%s\n", "Exit node", "no")
		}

		return nil
	}

	return command
}

func deleteMachineCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "delete",
		Short:        "Deletes a machine",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DeleteMachineRequest{MachineId: machineID}
		if _, err := tc.Client().DeleteMachine(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine deleted.")

		return nil
	}

	return command
}

func expireMachineCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "expire",
		Short:        "Expires a machine",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.ExpireMachineRequest{MachineId: machineID}
		if _, err := tc.Client().ExpireMachine(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine key expired.")

		return nil
	}

	return command
}

func authorizeMachineCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "authorize",
		Short:        "Authorizes a machine",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.AuthorizeMachineRequest{MachineId: machineID}
		if _, err := tc.Client().AuthorizeMachine(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine authorized.")

		return nil
	}

	return command
}

func listMachinesCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "list",
		Short:        "List machines",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.ListMachinesRequest{TailnetId: tc.TailnetID()}
		resp, err := tc.Client().ListMachines(cmd.Context(), connect.NewRequest(&req))

		if err != nil {
			return err
		}

		tbl := table.New("ID", "TAILNET", "NAME", "IPv4", "IPv6", "AUTHORIZED", "EPHEMERAL", "VERSION", "LAST_SEEN", "TAGS")
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
			tbl.AddRow(m.Id, m.Tailnet.Name, m.Name, m.Ipv4, m.Ipv6, m.Authorized, m.Ephemeral, m.ClientVersion, lastSeen, strings.Join(m.Tags, ","))
		}
		tbl.Print()

		return nil
	}

	return command
}

func getMachineRoutesCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "get-routes",
		Short:        "Show routes advertised and enabled by a given machine",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID.")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.GetMachineRoutesRequest{MachineId: machineID}
		resp, err := tc.Client().GetMachineRoutes(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printMachinesRoutesResponse(resp.Msg.Routes)

		return nil
	}

	return command
}

func enableMachineRoutesCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "enable-routes",
		Short:        "Enable routes for a given machine",
		SilenceUsage: true,
	})

	var machineID uint64
	var routes []string
	var replace bool

	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringSliceVar(&routes, "routes", []string{}, "List of routes to enable")
	command.Flags().BoolVar(&replace, "replace", false, "Replace current enabled routes with this new list")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		for _, r := range routes {
			if _, err := netaddr.ParseIPPrefix(r); err != nil {
				return err
			}
		}

		req := api.EnableMachineRoutesRequest{MachineId: machineID, Routes: routes, Replace: replace}
		resp, err := tc.Client().EnableMachineRoutes(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printMachinesRoutesResponse(resp.Msg.Routes)

		return nil
	}

	return command
}

func disableMachineRoutesCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "disable-routes",
		Short:        "Disable routes for a given machine",
		SilenceUsage: true,
	})

	var machineID uint64
	var routes []string

	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringSliceVar(&routes, "routes", []string{}, "List of routes to enable")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		for _, r := range routes {
			if _, err := netaddr.ParseIPPrefix(r); err != nil {
				return err
			}
		}

		req := api.DisableMachineRoutesRequest{MachineId: machineID, Routes: routes}
		resp, err := tc.Client().DisableMachineRoutes(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printMachinesRoutesResponse(resp.Msg.Routes)

		return nil
	}

	return command
}

func enableExitNodeCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "enable-exit-node",
		Short:        "Enable given machine as an exit node",
		SilenceUsage: true,
	})

	var machineID uint64
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.EnableExitNodeRequest{MachineId: machineID}
		resp, err := tc.Client().EnableExitNode(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printMachinesRoutesResponse(resp.Msg.Routes)

		return nil
	}

	return command
}

func disableExitNodeCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "disable-exit-node",
		Short:        "Disable given machine as an exit node",
		SilenceUsage: true,
	})

	var machineID uint64

	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DisableExitNodeRequest{MachineId: machineID}
		resp, err := tc.Client().DisableExitNode(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printMachinesRoutesResponse(resp.Msg.Routes)

		return nil
	}

	return command
}

func enableMachineKeyExpiryCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "enable-key-expiry",
		Short:        "Enable machine key expiry",
		SilenceUsage: true,
	}

	return configureSetMachineKeyExpiryCommand(command, false)
}

func disableMachineKeyExpiryCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "disable-key-expiry",
		Short:        "Disable machine key expiry",
		SilenceUsage: true,
	}

	return configureSetMachineKeyExpiryCommand(command, true)
}

func configureSetMachineKeyExpiryCommand(cmdTmpl *cobra.Command, disable bool) *cobra.Command {
	command, tc := prepareCommand(false, cmdTmpl)

	var machineID uint64

	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")

	_ = command.MarkFlagRequired("machine-id")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.SetMachineKeyExpiryRequest{MachineId: machineID, Disabled: disable}
		_, err := tc.Client().SetMachineKeyExpiry(cmd.Context(), connect.NewRequest(&req))
		if err != nil {
			return err
		}

		return nil
	}

	return command
}

func renameMachineCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "rename",
		Short:        "Rename given machine",
		SilenceUsage: true,
	})

	var machineID uint64
	var name string
	command.Flags().Uint64Var(&machineID, "machine-id", 0, "Machine ID")
	command.Flags().StringVar(&name, "new-name", "", "New name for the machine")

	_ = command.MarkFlagRequired("machine-id")
	_ = command.MarkFlagRequired("new-name")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.RenameMachineRequest{MachineId: machineID, MachineNewName: name}
		if _, err := tc.Client().RenameMachine(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		fmt.Println("Machine renamed to", name)

		return nil
	}

	return command
}

func printMachinesRoutesResponse(msg *api.MachineRoutes) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	for i, t := range msg.AdvertisedRoutes {
		if i == 0 {
			fmt.Fprintf(w, "%s\t%s\n", "Advertised routes", t)
		} else {
			fmt.Fprintf(w, "%s\t%s\n", "", t)
		}
	}

	for i, t := range msg.EnabledRoutes {
		if i == 0 {
			fmt.Fprintf(w, "%s\t%s\n", "Enabled routes", t)
		} else {
			fmt.Fprintf(w, "%s\t%s\n", "", t)
		}
	}

	if msg.AdvertisedExitNode {
		if msg.EnabledExitNode {
			fmt.Fprintf(w, "%s\t%s\n", "Exit node", "enabled")
		} else {
			fmt.Fprintf(w, "%s\t%s\n", "Exit node", "disabled")
		}
	} else {
		fmt.Fprintf(w, "%s\t%s\n", "Exit node", "no")
	}
}
