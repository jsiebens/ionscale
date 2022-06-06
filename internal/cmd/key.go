package cmd

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/muesli/coral"
)

func keyCommand() *coral.Command {
	command := &coral.Command{
		Use:          "genkey",
		SilenceUsage: true,
	}

	command.RunE = func(command *coral.Command, args []string) error {
		serverKey := key.NewServerKey()
		fmt.Println(serverKey.String())
		return nil
	}

	return command
}
