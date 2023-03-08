package cmd

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	rootCmd := rootCommand()
	rootCmd.AddCommand(configureCommand())
	rootCmd.AddCommand(keyCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(serverCommand())
	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(tailnetCommand())
	rootCmd.AddCommand(authkeysCommand())
	rootCmd.AddCommand(machineCommands())
	rootCmd.AddCommand(userCommands())
	rootCmd.AddCommand(systemCommand())

	return rootCmd
}

func Execute() error {
	return Command().Execute()
}

func rootCommand() *cobra.Command {
	return &cobra.Command{
		Use: "ionscale",
	}
}
