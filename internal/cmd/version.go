package cmd

import (
	"fmt"
	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/version"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	command, tc := prepareCommand(false, &cobra.Command{
		Use:          "version",
		Short:        "Display version information",
		SilenceUsage: true,
	})

	command.Run = func(cmd *cobra.Command, args []string) {
		clientVersion, clientRevision := version.GetReleaseInfo()
		fmt.Printf(`
Client:
 Version:       %s 
 Git Revision:  %s
`, clientVersion, clientRevision)

		resp, err := tc.Client().GetVersion(cmd.Context(), connect.NewRequest(&api.GetVersionRequest{}))
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
`, tc.Addr(), resp.Msg.Version, resp.Msg.Revision)

	}

	return command
}
