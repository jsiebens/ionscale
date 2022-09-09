package cmd

import (
	"errors"
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/muesli/coral"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

func configureCommand() *coral.Command {
	command := &coral.Command{
		Use:          "configure",
		Short:        "Generate a simple config file to get started.",
		SilenceUsage: true,
	}

	var domain string
	var acme bool
	var email string
	var dataDir string
	var certFile string
	var keyFile string

	command.Flags().StringVar(&domain, "domain", "", "Public domain name of your ionscale instance.")
	command.Flags().StringVar(&dataDir, "data-dir", "/var/lib/ionscale", "")
	command.Flags().BoolVar(&acme, "acme", false, "Get automatic certificate from Letsencrypt.org using ACME.")
	command.Flags().StringVar(&email, "acme-email", "", "Email to receive updates from Letsencrypt.org.")
	command.Flags().StringVar(&certFile, "cert-file", "", "Path to a TLS certificate file.")
	command.Flags().StringVar(&keyFile, "key-file", "", "Path to a TLS key file.")

	command.MarkFlagRequired("domain")

	command.PreRunE = func(cmd *coral.Command, args []string) error {
		if domain == "" {
			return errors.New("required flag 'domain' is missing")
		}

		if acme && email == "" {
			return errors.New("flag 'acme-email' is required when acme is enabled")
		}

		if !acme && (certFile == "" || keyFile == "") {
			return errors.New("flags 'cert-file' and 'key-file' are required when acme is disabled")
		}

		return nil
	}

	command.RunE = func(command *coral.Command, args []string) error {
		c := &config.Config{}

		c.HttpListenAddr = "0.0.0.0:80"
		c.HttpsListenAddr = "0.0.0.0:443"
		c.MetricsListenAddr = "127.0.0.1:9090"
		c.ServerUrl = fmt.Sprintf("https://%s", domain)

		c.Keys = config.Keys{
			ControlKey:       key.NewServerKey().String(),
			LegacyControlKey: key.NewServerKey().String(),
			SystemAdminKey:   key.NewServerKey().String(),
		}

		c.Tls = config.Tls{}
		if acme {
			c.Tls.AcmeEnabled = true
			c.Tls.AcmeEmail = email
			c.Tls.AcmePath = filepath.Join(dataDir, "acme")
		} else {
			c.Tls.CertFile = certFile
			c.Tls.KeyFile = keyFile
		}

		c.Database = config.Database{
			Type: "sqlite",
			Url:  filepath.Join(dataDir, "ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"),
		}

		configAsYaml, err := yaml.Marshal(c)
		if err != nil {
			return err
		}

		fmt.Println(string(configAsYaml))

		return nil
	}

	return command
}
