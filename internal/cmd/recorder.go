package cmd

import (
	"github.com/jsiebens/ionscale/pkg/ssh"
	"github.com/spf13/cobra"
)

func recorderCommand() *cobra.Command {
	t := ssh.RecorderConfig{}

	command := &cobra.Command{
		Use:          "recorder",
		Short:        "Start an SSH Recorder",
		SilenceUsage: true,
	}

	command.Flags().StringVar(&t.LoginServer, "login-server", "", "Base URL of control server")
	command.Flags().StringVar(&t.StateDir, "statedir", "", "Directory where the recorder should store its internal state")
	command.Flags().StringVar(&t.Dir, "dst", "", "Directory where recordings will be saved.")
	command.Flags().StringVar(&t.AuthKey, "auth-key", "", "")
	command.Flags().StringVar(&t.Hostname, "hostname", "recorder", "")

	command.RunE = func(command *cobra.Command, args []string) error {
		return ssh.Start(command.Context(), t)
	}

	return command
}
