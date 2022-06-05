package cmd

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/version"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/muesli/coral"
)

func versionCommand() *coral.Command {
	var command = &coral.Command{
		Use:          "version",
		Short:        "Display version information",
		SilenceUsage: true,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.Run = func(cmd *coral.Command, args []string) {
		clientVersion, clientRevision := version.GetReleaseInfo()
		fmt.Printf(`
Client:
 Version:       %s 
 Git Revision:  %s
`, clientVersion, clientRevision)

		client, err := target.createGRPCClient()
		if err != nil {
			fmt.Printf(`
Server:
 Error:         %s
`, err)
			return
		}

		resp, err := client.GetVersion(context.Background(), connect.NewRequest(&api.GetVersionRequest{}))
		if err != nil {
			fmt.Printf(`
Server:
 Error:         %s
`, err)
			return
		}

		fmt.Printf(`
Server:
 Addr:          %s
 Version:       %s 
 Git Revision:  %s
`, target.getAddr(), resp.Msg.Version, resp.Msg.Revision)

	}

	return command
}
