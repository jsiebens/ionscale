package cmd

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/server"
	"github.com/spf13/cobra"
)

func serverCommand() *cobra.Command {
	command := &cobra.Command{
		Use:           "server",
		Short:         "Start an ionscale server",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var configFile string

	command.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file.")

	command.RunE = func(command *cobra.Command, args []string) error {

		c, err := config.LoadConfig(configFile)
		if err != nil {
			return err
		}

		return server.Start(c)
	}

	return command
}
