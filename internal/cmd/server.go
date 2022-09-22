package cmd

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/server"
	"github.com/muesli/coral"
	"time"
)

func serverCommand() *coral.Command {
	command := &coral.Command{
		Use:          "server",
		Short:        "Start an ionscale server",
		SilenceUsage: true,
	}

	var cbf = configByFlags{}
	var configFile string

	cbf.prepareCommand(command)
	command.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file.")

	command.RunE = func(command *coral.Command, args []string) error {

		c, err := config.LoadConfig(configFile, &cbf.c)
		if err != nil {
			return err
		}

		return server.Start(c)
	}

	return command
}

type configByFlags struct {
	c config.Config
}

func (c *configByFlags) prepareCommand(cmd *coral.Command) {
	cmd.Flags().StringVar(&c.c.HttpListenAddr, "http-listen-addr", "", "")
	cmd.Flags().StringVar(&c.c.HttpsListenAddr, "https-listen-addr", "", "")
	cmd.Flags().StringVar(&c.c.MetricsListenAddr, "metrics-listen-addr", "", "")
	cmd.Flags().StringVar(&c.c.ServerUrl, "server-url", "", "")

	cmd.Flags().BoolVar(&c.c.Tls.Disable, "tls-disable", false, "")
	cmd.Flags().BoolVar(&c.c.Tls.ForceHttps, "tls-force-https", true, "")
	cmd.Flags().StringVar(&c.c.Tls.CertFile, "tls-cert-file", "", "")
	cmd.Flags().StringVar(&c.c.Tls.KeyFile, "tls-key-file", "", "")
	cmd.Flags().BoolVar(&c.c.Tls.AcmeEnabled, "tls-acme", false, "")
	cmd.Flags().StringVar(&c.c.Tls.AcmeEmail, "tls-acme-email", "", "")
	cmd.Flags().StringVar(&c.c.Tls.AcmeCA, "tls-acme-ca", "", "")
	cmd.Flags().StringVar(&c.c.Tls.AcmePath, "tls-acme-path", "", "")

	cmd.Flags().DurationVar(&c.c.PollNet.KeepAliveInterval, "poll-net-keep-alive-interval", 1*time.Minute, "")

	cmd.Flags().StringVar(&c.c.Keys.SystemAdminKey, "system-admin-key", "", "")
	cmd.Flags().StringVar(&c.c.Keys.ControlKey, "control-key", "", "")
	cmd.Flags().StringVar(&c.c.Keys.LegacyControlKey, "legacy-control-key", "", "")

	cmd.Flags().StringVar(&c.c.Database.Type, "database-type", "", "")
	cmd.Flags().StringVar(&c.c.Database.Url, "database-url", "", "")

	cmd.Flags().StringVar(&c.c.Logging.Level, "logging-level", "", "")
	cmd.Flags().StringVar(&c.c.Logging.Format, "logging-format", "", "")
	cmd.Flags().StringVar(&c.c.Logging.File, "logging-file", "", "")
}
