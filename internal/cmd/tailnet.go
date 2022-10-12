package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	idomain "github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"tailscale.com/tailcfg"
)

func tailnetCommand() *coral.Command {
	command := &coral.Command{
		Use:     "tailnets",
		Aliases: []string{"tailnet"},
		Short:   "Manage ionscale tailnets",
	}

	command.AddCommand(listTailnetsCommand())
	command.AddCommand(createTailnetsCommand())
	command.AddCommand(deleteTailnetCommand())
	command.AddCommand(getDNSConfigCommand())
	command.AddCommand(setDNSConfigCommand())
	command.AddCommand(getACLConfigCommand())
	command.AddCommand(setACLConfigCommand())
	command.AddCommand(editACLConfigCommand())
	command.AddCommand(getIAMPolicyCommand())
	command.AddCommand(setIAMPolicyCommand())
	command.AddCommand(editIAMPolicyCommand())
	command.AddCommand(enableHttpsCommand())
	command.AddCommand(disableHttpsCommand())
	command.AddCommand(enableServiceCollectionCommand())
	command.AddCommand(disableServiceCollectionCommand())
	command.AddCommand(enableFileSharingCommand())
	command.AddCommand(disableFileSharingCommand())
	command.AddCommand(enableSSHCommand())
	command.AddCommand(disableSSHCommand())
	command.AddCommand(getDERPMap())
	command.AddCommand(setDERPMap())
	command.AddCommand(resetDERPMap())

	return command
}

func listTailnetsCommand() *coral.Command {
	command := &coral.Command{
		Use:          "list",
		Short:        "List available Tailnets",
		SilenceUsage: true,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.RunE = func(command *coral.Command, args []string) error {

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

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
		Short:        "Create a new Tailnet",
		SilenceUsage: true,
	}

	var name string
	var domain string
	var email string
	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVarP(&name, "name", "n", "", "")
	command.Flags().StringVar(&domain, "domain", "", "")
	command.Flags().StringVar(&email, "email", "", "")

	command.PreRunE = func(cmd *coral.Command, args []string) error {
		if name == "" && email == "" && domain == "" {
			return fmt.Errorf("at least flag --name, --email or --domain is required")
		}
		if domain != "" && email != "" {
			return fmt.Errorf("flags --email and --domain are mutually exclusive")
		}
		return nil
	}

	command.RunE = func(command *coral.Command, args []string) error {

		var tailnetName = ""
		var iamPolicy = api.IAMPolicy{}

		if len(domain) != 0 {
			domainToLower := strings.ToLower(domain)
			tailnetName = domainToLower
			iamPolicy = api.IAMPolicy{
				Filters: []string{fmt.Sprintf("domain == %s", domainToLower)},
			}
		}

		if len(email) != 0 {
			emailToLower := strings.ToLower(email)
			tailnetName = emailToLower
			iamPolicy = api.IAMPolicy{
				Emails: []string{emailToLower},
				Roles: map[string]string{
					emailToLower: string(idomain.UserRoleAdmin),
				},
			}
		}

		if len(name) != 0 {
			tailnetName = name
		}

		client, err := target.createGRPCClient()
		if err != nil {
			return err
		}

		resp, err := client.CreateTailnet(context.Background(), connect.NewRequest(&api.CreateTailnetRequest{
			Name:      tailnetName,
			IamPolicy: &iamPolicy,
		}))

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

	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().BoolVar(&force, "force", false, "When enabled, force delete the specified Tailnet even when machines are still available.")

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

		_, err = client.DeleteTailnet(context.Background(), connect.NewRequest(&api.DeleteTailnetRequest{TailnetId: tailnet.Id, Force: force}))

		if err != nil {
			return err
		}

		fmt.Println("Tailnet deleted.")

		return nil
	}

	return command
}

func getDERPMap() *coral.Command {
	command := &coral.Command{
		Use:          "get-derp-map",
		Short:        "Get the DERP Map configuration",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var asJson bool

	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

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

		resp, err := client.GetDERPMap(context.Background(), connect.NewRequest(&api.GetDERPMapRequest{TailnetId: tailnet.Id}))

		if err != nil {
			return err
		}

		var derpMap struct {
			Regions map[int]*tailcfg.DERPRegion
		}

		if err := json.Unmarshal(resp.Msg.Value, &derpMap); err != nil {
			return err
		}

		if asJson {
			marshal, err := json.MarshalIndent(derpMap, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(derpMap)
			if err != nil {
				return err
			}

			fmt.Println(string(marshal))
		}

		return nil
	}

	return command
}

func setDERPMap() *coral.Command {
	command := &coral.Command{
		Use:          "set-derp-map",
		Short:        "Set the DERP Map configuration",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var file string
	var target = Target{}
	target.prepareCommand(command)

	command.Flags().StringVar(&tailnetName, "tailnet", "", "Tailnet name. Mutually exclusive with --tailnet-id.")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "Tailnet ID. Mutually exclusive with --tailnet.")
	command.Flags().StringVar(&file, "file", "", "Path to json file with the DERP Map configuration")

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

		rawJson, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		resp, err := client.SetDERPMap(context.Background(), connect.NewRequest(&api.SetDERPMapRequest{TailnetId: tailnet.Id, Value: rawJson}))
		if err != nil {
			return err
		}

		var derpMap tailcfg.DERPMap
		if err := json.Unmarshal(resp.Msg.Value, &derpMap); err != nil {
			return err
		}

		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}

func resetDERPMap() *coral.Command {
	command := &coral.Command{
		Use:          "reset-derp-map",
		Short:        "Reset the DERP Map to the default configuration",
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

		if _, err := client.ResetDERPMap(context.Background(), connect.NewRequest(&api.ResetDERPMapRequest{TailnetId: tailnet.Id})); err != nil {
			return err
		}

		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}

func enableFileSharingCommand() *coral.Command {
	command := &coral.Command{
		Use:          "enable-file-sharing",
		Aliases:      []string{"enable-taildrop"},
		Short:        "Enable Taildrop, the file sharing feature",
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

		req := api.EnableFileSharingRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.EnabledFileSharing(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableFileSharingCommand() *coral.Command {
	command := &coral.Command{
		Use:          "disable-file-sharing",
		Aliases:      []string{"disable-taildrop"},
		Short:        "Disable Taildrop, the file sharing feature",
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

		req := api.DisableFileSharingRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.DisableFileSharing(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func enableServiceCollectionCommand() *coral.Command {
	command := &coral.Command{
		Use:          "enable-service-collection",
		Short:        "Enable monitoring live services running on your network’s machines.",
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

		req := api.EnableServiceCollectionRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.EnabledServiceCollection(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableServiceCollectionCommand() *coral.Command {
	command := &coral.Command{
		Use:          "disable-service-collection",
		Short:        "Disable monitoring live services running on your network’s machines.",
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

		req := api.DisableServiceCollectionRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.DisableServiceCollection(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func enableSSHCommand() *coral.Command {
	command := &coral.Command{
		Use:          "enable-ssh",
		Short:        "Enable ssh access using tailnet and ACLs.",
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

		req := api.EnableSSHRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.EnabledSSH(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableSSHCommand() *coral.Command {
	command := &coral.Command{
		Use:          "disable-ssh",
		Short:        "Disable ssh access using tailnet and ACLs.",
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

		req := api.DisableSSHRequest{
			TailnetId: tailnet.Id,
		}

		if _, err := client.DisableSSH(context.Background(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}
