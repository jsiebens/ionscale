package cmd

import (
	"github.com/muesli/coral"
)

func Command() *coral.Command {
	rootCmd := rootCommand()
	rootCmd.AddCommand(configureCommand())
	rootCmd.AddCommand(keyCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(derpMapCommand())
	rootCmd.AddCommand(serverCommand())
	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(tailnetCommand())
	rootCmd.AddCommand(authkeysCommand())
	rootCmd.AddCommand(machineCommands())
	rootCmd.AddCommand(userCommands())

	return rootCmd
}

func Execute() error {
	return Command().Execute()
}

func rootCommand() *coral.Command {
	return &coral.Command{
		Use: "ionscale",
	}
}
