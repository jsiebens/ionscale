package cmd

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/internal/version"
	"github.com/jsiebens/ionscale/pkg/gen/api"
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

		client, c, err := target.createGRPCClient()
		if err != nil {
			fmt.Printf(`
Server:
 Error:         %s
`, err)
			return
		}
		defer safeClose(c)

		resp, err := client.GetVersion(context.Background(), &api.GetVersionRequest{})
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
`, target.getAddr(), resp.Version, resp.Revision)

	}

	return command
}
