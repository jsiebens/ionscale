package cmd

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
	str2dur "github.com/xhit/go-str2duration/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"strings"
	"time"
)

func authkeysCommand() *coral.Command {
	command := &coral.Command{
		Use: "auth-keys",
	}

	command.AddCommand(createAuthkeysCommand())
	command.AddCommand(deleteAuthKeyCommand())
	command.AddCommand(listAuthkeysCommand())

	return command
}

func createAuthkeysCommand() *coral.Command {
	command := &coral.Command{
		Use:          "create",
		SilenceUsage: true,
	}

	var tailnetID uint64
	var tailnetName string
	var ephemeral bool
	var tags []string
	var expiry string
	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVar(&tailnetName, "tailnet", "", "")
	command.Flags().Uint64Var(&tailnetID, "tailnet-id", 0, "")
	command.Flags().BoolVar(&ephemeral, "ephemeral", false, "")
	command.Flags().StringSliceVar(&tags, "tag", []string{}, "")
	command.Flags().StringVar(&expiry, "expiry", "180d", "")

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

		var expiryDur *durationpb.Duration

		if expiry != "" && expiry != "none" {
			duration, err := str2dur.ParseDuration(expiry)
			if err != nil {
				return err
			}
			expiryDur = durationpb.New(duration)
		}

		req := &api.CreateAuthKeyRequest{
			TailnetId: tailnet.Id,
			Ephemeral: ephemeral,
			Tags:      tags,
			Expiry:    expiryDur,
		}
		resp, err := client.CreateAuthKey(context.Background(), req)

		if err != nil {
			return err
		}

		fmt.Println("")
		fmt.Println("Generated new auth key")
		fmt.Println("Be sure to copy your new key below. It won't be shown in full again.")
		fmt.Println("")
		fmt.Printf("  %s\n", resp.Value)
		fmt.Println("")

		return nil
	}

	return command
}

func deleteAuthKeyCommand() *coral.Command {
	command := &coral.Command{
		Use:          "delete",
		SilenceUsage: true,
	}

	var authKeyId uint64
	var target = Target{}
	target.prepareCommand(command)
	command.Flags().Uint64Var(&authKeyId, "id", 0, "")

	command.RunE = func(command *coral.Command, args []string) error {
		grpcClient, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		req := api.DeleteAuthKeyRequest{AuthKeyId: authKeyId}
		if _, err := grpcClient.DeleteAuthKey(context.Background(), &req); err != nil {
			return err
		}

		fmt.Println("Auth key deleted.")

		return nil
	}

	return command
}

func listAuthkeysCommand() *coral.Command {
	command := &coral.Command{
		Use:          "list",
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

		req := &api.ListAuthKeysRequest{TailnetId: tailnet.Id}
		resp, err := client.ListAuthKeys(context.Background(), req)

		if err != nil {
			return err
		}

		printAuthKeyTable(resp.AuthKeys...)

		return nil
	}

	return command
}

func printAuthKeyTable(authKeys ...*api.AuthKey) {
	tbl := table.New("ID", "KEY", "EPHEMERAL", "EXPIRED", "EXPIRES_AT", "TAGS")
	for _, authKey := range authKeys {
		addAuthKeyToTable(tbl, authKey)
	}
	tbl.Print()
}

func addAuthKeyToTable(tbl table.Table, authKey *api.AuthKey) {
	var expired = false
	var expiresAt = "never"
	if authKey.ExpiresAt != nil {
		expiresAt = authKey.ExpiresAt.AsTime().Local().Format("2006-01-02 15:04:05")
		expired = time.Now().After(authKey.ExpiresAt.AsTime())
	}
	tbl.AddRow(authKey.Id, fmt.Sprintf("%s...", authKey.Key), authKey.Ephemeral, expired, expiresAt, strings.Join(authKey.Tags, ","))
}
