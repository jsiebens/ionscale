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
		Use:   "tailnets",
		Short: "Manage ionscale tailnets",
	}

	command.AddCommand(listTailnetsCommand())
	command.AddCommand(createTailnetsCommand())
	command.AddCommand(deleteTailnetCommand())
	command.AddCommand(getDNSConfigCommand())
	command.AddCommand(setDNSConfigCommand())
	command.AddCommand(getACLConfigCommand())
	command.AddCommand(setACLConfigCommand())
	command.AddCommand(getIAMPolicyCommand())
	command.AddCommand(setIAMPolicyCommand())
	command.AddCommand(enableHttpsCommand())
	command.AddCommand(disableHttpsCommand())
	command.AddCommand(getDERPMap())
	command.AddCommand(setDERPMap())

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

			fmt.Println()
			fmt.Println(string(marshal))
		} else {
			marshal, err := yaml.Marshal(derpMap)
			if err != nil {
				return err
			}

			fmt.Println()
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

		fmt.Println()
		fmt.Println("DERP Map updated successfully")

		return nil
	}

	return command
}
