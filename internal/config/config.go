package config

import (
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/caddyserver/certmagic"
	"github.com/imdario/mergo"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	tkey "tailscale.com/types/key"
	"time"
)

const (
	defaultKeepAliveInterval = 1 * time.Minute
	defaultMagicDNSSuffix    = "ionscale.net"
)

var (
	keepAliveInterval = defaultKeepAliveInterval
	magicDNSSuffix    = defaultMagicDNSSuffix
	certDNSSuffix     = ""
)

func KeepAliveInterval() time.Duration {
	return keepAliveInterval
}

func MagicDNSSuffix() string {
	return magicDNSSuffix
}

func CertDNSSuffix() string {
	return certDNSSuffix
}

func LoadConfig(path string) (*Config, error) {
	cfg := defaultConfig()

	if len(path) != 0 {
		expandedPath, err := homedir.Expand(path)
		if err != nil {
			return nil, err
		}

		absPath, err := filepath.Abs(expandedPath)
		if err != nil {
			return nil, err
		}

		b, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
		}
	}

	envCfg := &Config{}
	if err := env.Parse(envCfg, env.Options{Prefix: "IONSCALE_"}); err != nil {
		return nil, err
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(cfg, envCfg, mergo.WithOverride); err != nil {
		return nil, err
	}

	keepAliveInterval = cfg.PollNet.KeepAliveInterval
	magicDNSSuffix = cfg.DNS.MagicDNSSuffix

	if cfg.DNS.Provider.Zone != "" {
		if cfg.DNS.Provider.Subdomain == "" {
			certDNSSuffix = cfg.DNS.Provider.Zone
		} else {
			certDNSSuffix = fmt.Sprintf("%s.%s", cfg.DNS.Provider.Subdomain, cfg.DNS.Provider.Zone)
		}
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		HttpListenAddr:    ":8080",
		HttpsListenAddr:   ":8443",
		MetricsListenAddr: ":9091",
		ServerUrl:         "https://localhost:8843",
		Database: Database{
			Type: "sqlite",
			Url:  "./ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
		},
		Tls: Tls{
			Disable:     false,
			ForceHttps:  true,
			AcmeEnabled: false,
			AcmeCA:      certmagic.LetsEncryptProductionCA,
			AcmePath:    "./acme",
		},
		PollNet: PollNet{
			KeepAliveInterval: defaultKeepAliveInterval,
		},
		DNS: DNS{
			MagicDNSSuffix: defaultMagicDNSSuffix,
		},
		Logging: Logging{
			Level: "info",
		},
	}
}

type ServerKeys struct {
	SystemAdminKey   *key.ServerPrivate
	ControlKey       tkey.MachinePrivate
	LegacyControlKey tkey.MachinePrivate
}

type Config struct {
	HttpListenAddr    string   `yaml:"http_listen_addr,omitempty" env:"HTTP_LISTEN_ADDR"`
	HttpsListenAddr   string   `yaml:"https_listen_addr,omitempty" env:"HTTPS_LISTEN_ADDR"`
	MetricsListenAddr string   `yaml:"metrics_listen_addr,omitempty" env:"METRICS_LISTEN_ADDR"`
	ServerUrl         string   `yaml:"server_url,omitempty" env:"SERVER_URL"`
	Tls               Tls      `yaml:"tls,omitempty" envPrefix:"TLS_"`
	PollNet           PollNet  `yaml:"poll_net,omitempty" envPrefix:"POLL_NET_"`
	Keys              Keys     `yaml:"keys,omitempty" envPrefix:"KEYS_"`
	Database          Database `yaml:"database,omitempty" envPrefix:"DB_"`
	Auth              Auth     `yaml:"auth,omitempty" envPrefix:"AUTH_"`
	DNS               DNS      `yaml:"dns,omitempty"`
	Logging           Logging  `yaml:"logging,omitempty" envPrefix:"LOGGING_"`
}

type Tls struct {
	Disable     bool   `yaml:"disable" env:"DISABLE"`
	ForceHttps  bool   `yaml:"force_https" env:"FORCE_HTTPS"`
	CertFile    string `yaml:"cert_file,omitempty" env:"CERT_FILE"`
	KeyFile     string `yaml:"key_file,omitempty" env:"KEY_FILE"`
	AcmeEnabled bool   `yaml:"acme,omitempty" env:"ACME_ENABLED"`
	AcmeEmail   string `yaml:"acme_email,omitempty" env:"ACME_EMAIL"`
	AcmeCA      string `yaml:"acme_ca,omitempty" env:"ACME_CA"`
	AcmePath    string `yaml:"acme_path,omitempty" env:"ACME_PATH"`
}

type PollNet struct {
	KeepAliveInterval time.Duration `yaml:"keep_alive_interval" env:"KEEP_ALIVE_INTERVAL"`
}

type Logging struct {
	Level  string `yaml:"level,omitempty" env:"LEVEL"`
	Format string `yaml:"format,omitempty" env:"FORMAT"`
	File   string `yaml:"file,omitempty" env:"FILE"`
}

type Database struct {
	Type string `yaml:"type,omitempty" env:"TYPE"`
	Url  string `yaml:"url,omitempty" env:"URL"`
}

type Keys struct {
	ControlKey       string `yaml:"control_key,omitempty" env:"CONTROL_KEY"`
	LegacyControlKey string `yaml:"legacy_control_key,omitempty" env:"LEGACY_CONTROL_KEY"`
	SystemAdminKey   string `yaml:"system_admin_key,omitempty" env:"SYSTEM_ADMIN_KEY"`
}

type Auth struct {
	Provider          AuthProvider      `yaml:"provider,omitempty" envPrefix:"PROVIDER_"`
	SystemAdminPolicy SystemAdminPolicy `yaml:"system_admins"`
}

type AuthProvider struct {
	Issuer       string   `yaml:"issuer" env:"ISSUER"`
	ClientID     string   `yaml:"client_id" env:"CLIENT_ID"`
	ClientSecret string   `yaml:"client_secret" env:"CLIENT_SECRET"`
	Scopes       []string `yaml:"additional_scopes"  env:"SCOPES"`
}

type DNS struct {
	MagicDNSSuffix string      `yaml:"magic_dns_suffix"`
	Provider       DNSProvider `yaml:"provider,omitempty"`
}

type DNSProvider struct {
	Name          string            `yaml:"name"`
	Zone          string            `yaml:"zone"`
	Subdomain     string            `yaml:"subdomain"`
	Configuration map[string]string `yaml:"config"`
}

type SystemAdminPolicy struct {
	Subs    []string `yaml:"subs,omitempty"`
	Emails  []string `yaml:"emails,omitempty"`
	Filters []string `yaml:"filters,omitempty"`
}

func (c *Config) CreateUrl(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	return strings.TrimSuffix(c.ServerUrl, "/") + "/" + strings.TrimPrefix(path, "/")
}

func (c *Config) ReadServerKeys(defaultKeys *domain.ControlKeys) (*ServerKeys, error) {
	keys := &ServerKeys{
		ControlKey:       defaultKeys.ControlKey,
		LegacyControlKey: defaultKeys.LegacyControlKey,
	}

	if len(c.Keys.SystemAdminKey) != 0 {
		systemAdminKey, err := key.ParsePrivateKey(c.Keys.SystemAdminKey)
		if err != nil {
			return nil, fmt.Errorf("error reading system admin key: %v", err)
		}

		keys.SystemAdminKey = systemAdminKey
	}

	if len(c.Keys.ControlKey) != 0 {
		controlKey, err := util.ParseMachinePrivateKey(c.Keys.ControlKey)
		if err != nil {
			return nil, fmt.Errorf("error reading control key: %v", err)
		}
		keys.ControlKey = *controlKey
	}

	if len(c.Keys.LegacyControlKey) != 0 {
		legacyControlKey, err := util.ParseMachinePrivateKey(c.Keys.LegacyControlKey)
		if err != nil {
			return nil, fmt.Errorf("error reading legacy control key: %v", err)
		}
		keys.LegacyControlKey = *legacyControlKey
	}

	return keys, nil
}
