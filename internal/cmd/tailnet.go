package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bufbuild/connect-go"
	idomain "github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/pkg/defaults"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"tailscale.com/tailcfg"
)

func tailnetCommand() *cobra.Command {
	command := &cobra.Command{
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
	command.AddCommand(enableServiceCollectionCommand())
	command.AddCommand(disableServiceCollectionCommand())
	command.AddCommand(enableFileSharingCommand())
	command.AddCommand(disableFileSharingCommand())
	command.AddCommand(enableSSHCommand())
	command.AddCommand(disableSSHCommand())
	command.AddCommand(enableMachineAuthorizationCommand())
	command.AddCommand(disableMachineAuthorizationCommand())
	command.AddCommand(getDERPMap())
	command.AddCommand(setDERPMap())
	command.AddCommand(resetDERPMap())

	return command
}

func listTailnetsCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "list",
		Short:        "List available Tailnets",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		resp, err := tc.Client().ListTailnets(cmd.Context(), connect.NewRequest(&api.ListTailnetsRequest{}))

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

func createTailnetsCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "create",
		Short:        "Create a new Tailnet",
		SilenceUsage: true,
	})

	var name string
	var domain string
	var email string

	command.Flags().StringVarP(&name, "name", "n", "", "")
	command.Flags().StringVar(&domain, "domain", "", "")
	command.Flags().StringVar(&email, "email", "", "")

	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if name == "" {
			return fmt.Errorf("flag --name is required")
		}
		if domain != "" && email != "" {
			return fmt.Errorf("flags --email and --domain are mutually exclusive")
		}
		return nil
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {

		dnsConfig := defaults.DefaultDNSConfig()
		aclPolicy := defaults.DefaultACLPolicy()
		iamPolicy := &api.IAMPolicy{}

		if len(domain) != 0 {
			domainToLower := strings.ToLower(domain)
			iamPolicy = &api.IAMPolicy{
				Filters: []string{fmt.Sprintf("domain == %s", domainToLower)},
			}
		}

		if len(email) != 0 {
			emailToLower := strings.ToLower(email)
			iamPolicy = &api.IAMPolicy{
				Emails: []string{emailToLower},
				Roles: map[string]string{
					emailToLower: string(idomain.UserRoleAdmin),
				},
			}
		}

		resp, err := tc.Client().CreateTailnet(cmd.Context(), connect.NewRequest(&api.CreateTailnetRequest{
			Name:      name,
			IamPolicy: iamPolicy,
			AclPolicy: aclPolicy,
			DnsConfig: dnsConfig,
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

func deleteTailnetCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "delete",
		Short:        "Delete a tailnet",
		SilenceUsage: true,
	})

	var force bool

	command.Flags().BoolVar(&force, "force", false, "When enabled, force delete the specified Tailnet even when machines are still available.")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := tc.Client().DeleteTailnet(cmd.Context(), connect.NewRequest(&api.DeleteTailnetRequest{TailnetId: tc.TailnetID(), Force: force}))
		if err != nil {
			return err
		}

		fmt.Println("Tailnet deleted.")

		return nil
	}

	return command
}

func getDERPMap() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "get-derp-map",
		Short:        "Get the DERP Map configuration",
		SilenceUsage: true,
	})

	var asJson bool

	command.Flags().BoolVar(&asJson, "json", false, "When enabled, render output as json otherwise yaml")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		resp, err := tc.Client().GetDERPMap(cmd.Context(), connect.NewRequest(&api.GetDERPMapRequest{TailnetId: tc.TailnetID()}))

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

func setDERPMap() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "set-derp-map",
		Short:        "Set the DERP Map configuration",
		SilenceUsage: true,
	})

	var file string

	command.Flags().StringVar(&file, "file", "", "Path to json file with the DERP Map configuration")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		rawJson, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		resp, err := tc.Client().SetDERPMap(cmd.Context(), connect.NewRequest(&api.SetDERPMapRequest{TailnetId: tc.TailnetID(), Value: rawJson}))
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

func resetDERPMap() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "reset-derp-map",
		Short:        "Reset the DERP Map to the default configuration",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		if _, err := tc.Client().ResetDERPMap(cmd.Context(), connect.NewRequest(&api.ResetDERPMapRequest{TailnetId: tc.TailnetID()})); err != nil {
			return err
		}

		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}

func enableFileSharingCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "enable-file-sharing",
		Aliases:      []string{"enable-taildrop"},
		Short:        "Enable Taildrop, the file sharing feature",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.EnableFileSharingRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().EnableFileSharing(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableFileSharingCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "disable-file-sharing",
		Aliases:      []string{"disable-taildrop"},
		Short:        "Disable Taildrop, the file sharing feature",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DisableFileSharingRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().DisableFileSharing(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func enableServiceCollectionCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "enable-service-collection",
		Short:        "Enable monitoring live services running on your network’s machines.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.EnableServiceCollectionRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().EnableServiceCollection(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableServiceCollectionCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "disable-service-collection",
		Short:        "Disable monitoring live services running on your network’s machines.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DisableServiceCollectionRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().DisableServiceCollection(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func enableSSHCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "enable-ssh",
		Short:        "Enable ssh access using tailnet and ACLs.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.EnableSSHRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().EnableSSH(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableSSHCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "disable-ssh",
		Short:        "Disable ssh access using tailnet and ACLs.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DisableSSHRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().DisableSSH(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func enableMachineAuthorizationCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "enable-machine-authorization",
		Short:        "Enable machine authorization.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.EnableMachineAuthorizationRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().EnableMachineAuthorization(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}

func disableMachineAuthorizationCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "disable-machine-authorization",
		Short:        "Disable machine authorization.",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.DisableMachineAuthorizationRequest{
			TailnetId: tc.TailnetID(),
		}

		if _, err := tc.Client().DisableMachineAuthorization(cmd.Context(), connect.NewRequest(&req)); err != nil {
			return err
		}

		return nil
	}

	return command
}
