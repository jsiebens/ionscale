package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"os"
	"strings"
	"text/tabwriter"
)

func getDNSConfigCommand() *coral.Command {
	command := &coral.Command{
		Use:          "get-dns",
		Short:        "Get DNS configuration",
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

		req := api.GetDNSConfigRequest{TailnetId: tailnet.Id}
		resp, err := client.GetDNSConfig(context.Background(), connect.NewRequest(&req))

		if err != nil {
			return err
		}
		config := resp.Msg.Config

		var allNameservers = config.Nameservers

		for i, j := range config.Routes {
			for _, n := range j.Routes {
				allNameservers = append(allNameservers, fmt.Sprintf("%s:%s", i, n))
			}
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 8, 8, 0, '\t', 0)
		defer w.Flush()

		fmt.Fprintf(w, "%s\t\t%v\n", "Override Local DNS", config.OverrideLocalDns)

		if config.MagicDns {
			fmt.Fprintf(w, "MagicDNS\t%s\t%s\n", config.MagicDnsSuffix, "100.100.100.100")
		}

		for k, r := range config.Routes {
			for i, t := range r.Routes {
				if i == 0 {
					fmt.Fprintf(w, "SplitDNS\t%s\t%s\n", k, t)
				} else {
					fmt.Fprintf(w, "%s\t%s\n", "", t)
				}
			}
		}

		for i, t := range config.Nameservers {
			if i == 0 {
				fmt.Fprintf(w, "%s\t%s\t%s\n", "Global", "", t)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\n", "", "", t)
			}
		}

		return nil
	}

	return command
}

func setDNSConfigCommand() *coral.Command {
	command := &coral.Command{
		Use:          "set-dns",
		Short:        "Set DNS config",
		SilenceUsage: true,
	}

	var nameservers []string
	var magicDNS bool
	var overrideLocalDNS bool
	var tailnetID uint64
	var tailnetName string
	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().StringSliceVarP(&nameservers, "nameserver", "", []string{}, "Machines on your network will use these nameservers to resolve DNS queries.")
	command.Flags().BoolVarP(&magicDNS, "magic-dns", "", false, "Enable MagicDNS for the specified Tailnet")
	command.Flags().BoolVarP(&overrideLocalDNS, "override-local-dns", "", false, "When enabled, connected clients ignore local DNS settings and always use the nameservers specified for this Tailnet")

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

		var globalNameservers []string
		var routes = make(map[string]*api.Routes)

		for _, n := range nameservers {
			split := strings.Split(n, ":")
			if len(split) == 2 {
				r, ok := routes[split[0]]
				if ok {
					r.Routes = append(r.Routes, split[1])
				} else {
					routes[split[0]] = &api.Routes{Routes: []string{split[1]}}
				}
			} else {
				globalNameservers = append(globalNameservers, n)
			}
		}

		req := api.SetDNSConfigRequest{
			TailnetId: tailnet.Id,
			Config: &api.DNSConfig{
				MagicDns:         magicDNS,
				OverrideLocalDns: overrideLocalDNS,
				Nameservers:      globalNameservers,
				Routes:           routes,
			},
		}
		resp, err := client.SetDNSConfig(context.Background(), connect.NewRequest(&req))

		if err != nil {
			return err
		}

		config := resp.Msg.Config

		var allNameservers = config.Nameservers

		for i, j := range config.Routes {
			for _, n := range j.Routes {
				allNameservers = append(allNameservers, fmt.Sprintf("%s:%s", i, n))
			}
		}

		fmt.Printf("%-*v%v\n", 25, "Magic DNS Enabled:", config.MagicDns)
		fmt.Printf("%-*v%v\n", 25, "Override Local DNS:", config.OverrideLocalDns)
		fmt.Printf("%-*v%v\n", 25, "Nameservers:", strings.Join(allNameservers, ","))

		return nil
	}

	return command
}
