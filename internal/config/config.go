package config

import (
	"encoding/base64"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/caddyserver/certmagic"
	"github.com/imdario/mergo"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"path/filepath"
	tkey "tailscale.com/types/key"
	"time"
)

const (
	defaultKeepAliveInterval = 1 * time.Minute
	defaultMagicDNSSuffix    = "ionscale.net"
)

var (
	keepAliveInterval     = defaultKeepAliveInterval
	magicDNSSuffix        = defaultMagicDNSSuffix
	dnsProviderConfigured = false
)

func KeepAliveInterval() time.Duration {
	return keepAliveInterval
}

func MagicDNSSuffix() string {
	return magicDNSSuffix
}

func DNSProviderConfigured() bool {
	return dnsProviderConfigured
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

	envCfgB64 := os.Getenv("IONSCALE_CONFIG_BASE64")
	if len(envCfgB64) != 0 {
		b, err := base64.StdEncoding.DecodeString(envCfgB64)
		if err != nil {
			return nil, err
		}

		// merge env configuration on top of the default/file configuration
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
		dnsProviderConfigured = true
	}

	return cfg.Validate()
}

func defaultConfig() *Config {
	return &Config{
		WebListenAddr:     ":8080",
		MetricsListenAddr: ":9091",
		Database: Database{
			Type:         "sqlite",
			Url:          "./ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
			MaxOpenConns: 0,
			MaxIdleConns: 2,
		},
		Tls: Tls{
			Disable:     false,
			ForceHttps:  true,
			AcmeEnabled: false,
			AcmeCA:      certmagic.LetsEncryptProductionCA,
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
		Events: Events{
			Log: EventsLogSink{
				Enabled: false,
			},
			File: EventsFileSink{
				Enabled:  false,
				FileName: "events.log",
			},
		},
	}
}

type ServerKeys struct {
	SystemAdminKey   *key.ServerPrivate
	ControlKey       tkey.MachinePrivate
	LegacyControlKey tkey.MachinePrivate
}

type Config struct {
	WebListenAddr     string   `yaml:"web_listen_addr,omitempty" env:"WEB_LISTEN_ADDR"`
	MetricsListenAddr string   `yaml:"metrics_listen_addr,omitempty" env:"METRICS_LISTEN_ADDR"`
	WebPublicAddr     string   `yaml:"web_public_addr,omitempty" env:"WEB_PUBLIC_ADDR"`
	Tls               Tls      `yaml:"tls,omitempty" envPrefix:"TLS_"`
	PollNet           PollNet  `yaml:"poll_net,omitempty" envPrefix:"POLL_NET_"`
	Keys              Keys     `yaml:"keys,omitempty" envPrefix:"KEYS_"`
	Database          Database `yaml:"database,omitempty" envPrefix:"DB_"`
	Auth              Auth     `yaml:"auth,omitempty" envPrefix:"AUTH_"`
	DNS               DNS      `yaml:"dns,omitempty"`
	Logging           Logging  `yaml:"logging,omitempty" envPrefix:"LOGGING_"`
	Events            Events   `yaml:"events,omitempty" envPrefix:"EVENTS_"`

	WebPublicUrl *url.URL `yaml:"-"`
}

type Tls struct {
	Disable     bool   `yaml:"disable" env:"DISABLE"`
	ForceHttps  bool   `yaml:"force_https" env:"FORCE_HTTPS"`
	CertFile    string `yaml:"cert_file,omitempty" env:"CERT_FILE"`
	KeyFile     string `yaml:"key_file,omitempty" env:"KEY_FILE"`
	AcmeEnabled bool   `yaml:"acme,omitempty" env:"ACME_ENABLED"`
	AcmeEmail   string `yaml:"acme_email,omitempty" env:"ACME_EMAIL"`
	AcmeCA      string `yaml:"acme_ca,omitempty" env:"ACME_CA"`
}

type PollNet struct {
	KeepAliveInterval time.Duration `yaml:"keep_alive_interval" env:"KEEP_ALIVE_INTERVAL"`
}

type Logging struct {
	Level  string `yaml:"level,omitempty" env:"LEVEL"`
	Format string `yaml:"format,omitempty" env:"FORMAT"`
	File   string `yaml:"file,omitempty" env:"FILE"`
}

type Events struct {
	Log  EventsLogSink  `yaml:"log,omitempty" envPrefix:"LOG_"`
	File EventsFileSink `yaml:"file,omitempty" envPrefix:"FILE_"`
	Tcp  EventsTcpSink  `yaml:"tcp,omitempty" envPrefix:"TCP_"`
}

type EventsLogSink struct {
	Enabled bool `yaml:"enabled,omitempty" env:"ENABLED"`
}

type EventsFileSink struct {
	Enabled     bool          `yaml:"enabled,omitempty" env:"ENABLED"`
	Path        string        `yaml:"path,omitempty" env:"PATH"`
	FileName    string        `yaml:"name,omitempty" env:"NAME"`
	MaxBytes    int           `yaml:"max_bytes,omitempty" env:"MAX_BYTES"`
	MaxDuration time.Duration `yaml:"max_duration,omitempty" env:"MAX_DURATION"`
	MaxFiles    int           `yaml:"max_files,omitempty" env:"MAX_FILES"`
}

type EventsTcpSink struct {
	Enabled bool   `yaml:"enabled,omitempty" env:"ENABLED"`
	Addr    string `yaml:"addr,omitempty" env:"ADDR"`
}

type Database struct {
	Type            string        `yaml:"type,omitempty" env:"TYPE"`
	Url             string        `yaml:"url,omitempty" env:"URL"`
	MaxOpenConns    int           `yaml:"max_open_conns,omitempty" env:"MAX_OPEN_CONNS"`
	MaxIdleConns    int           `yaml:"max_idle_conns,omitempty" env:"MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_life_time,omitempty" env:"CONN_MAX_LIFE_TIME"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time,omitempty" env:"CONN_MAX_IDLE_TIME"`
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
	Configuration map[string]string `yaml:"config"`
}

type SystemAdminPolicy struct {
	Subs    []string `yaml:"subs,omitempty"`
	Emails  []string `yaml:"emails,omitempty"`
	Filters []string `yaml:"filters,omitempty"`
}

func (c *Config) Validate() (*Config, error) {
	publicWebUrl, err := publicAddrToUrl(c.WebPublicAddr)
	if err != nil {
		return nil, err
	}

	c.WebPublicUrl = publicWebUrl
	return c, nil
}

func (c *Config) CreateUrl(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	u := url.URL{
		Scheme: c.WebPublicUrl.Scheme,
		Host:   c.WebPublicUrl.Host,
		Path:   path,
	}
	return u.String()
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
