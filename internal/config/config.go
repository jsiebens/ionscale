package config

import (
	"encoding/base64"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"tailscale.com/tailcfg"
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

		b, err = expandEnvVars(b)
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

		b, err = expandEnvVars(b)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
		}
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
		ListenAddr:        ":8080",
		MetricsListenAddr: ":9091",
		StunListenAddr:    ":3478",
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
		DERP: DERP{
			Server: DERPServer{
				Disabled:   false,
				RegionID:   1000,
				RegionCode: "ionscale",
				RegionName: "ionscale Embedded DERP",
			},
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
	ListenAddr        string   `yaml:"listen_addr,omitempty"`
	StunListenAddr    string   `yaml:"stun_listen_addr,omitempty"`
	MetricsListenAddr string   `yaml:"metrics_listen_addr,omitempty"`
	PublicAddr        string   `yaml:"public_addr,omitempty"`
	StunPublicAddr    string   `yaml:"stun_public_addr,omitempty"`
	Tls               Tls      `yaml:"tls,omitempty"`
	PollNet           PollNet  `yaml:"poll_net,omitempty"`
	Keys              Keys     `yaml:"keys,omitempty"`
	Database          Database `yaml:"database,omitempty"`
	Auth              Auth     `yaml:"auth,omitempty"`
	DNS               DNS      `yaml:"dns,omitempty"`
	DERP              DERP     `yaml:"derp,omitempty"`
	Logging           Logging  `yaml:"logging,omitempty"`

	PublicUrl *url.URL `yaml:"-"`

	stunHost string
	stunPort int
	derpHost string
	derpPort int
}

type Tls struct {
	Disable     bool   `yaml:"disable"`
	ForceHttps  bool   `yaml:"force_https"`
	CertFile    string `yaml:"cert_file,omitempty"`
	KeyFile     string `yaml:"key_file,omitempty"`
	AcmeEnabled bool   `yaml:"acme,omitempty"`
	AcmeEmail   string `yaml:"acme_email,omitempty"`
	AcmeCA      string `yaml:"acme_ca,omitempty"`
}

type PollNet struct {
	KeepAliveInterval time.Duration `yaml:"keep_alive_interval"`
}

type Logging struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`
	File   string `yaml:"file,omitempty"`
}

type Database struct {
	Type            string        `yaml:"type,omitempty"`
	Url             string        `yaml:"url,omitempty"`
	MaxOpenConns    int           `yaml:"max_open_conns,omitempty"`
	MaxIdleConns    int           `yaml:"max_idle_conns,omitempty"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_life_time,omitempty"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time,omitempty"`
}

type Keys struct {
	ControlKey       string `yaml:"control_key,omitempty"`
	LegacyControlKey string `yaml:"legacy_control_key,omitempty"`
	SystemAdminKey   string `yaml:"system_admin_key,omitempty"`
}

type Auth struct {
	Provider          AuthProvider      `yaml:"provider,omitempty"`
	SystemAdminPolicy SystemAdminPolicy `yaml:"system_admins"`
}

type AuthProvider struct {
	Issuer       string   `yaml:"issuer"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Scopes       []string `yaml:"additional_scopes" `
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

type DERP struct {
	Server  DERPServer `yaml:"server,omitempty"`
	Sources []string   `yaml:"sources,omitempty"`
}

type DERPServer struct {
	Disabled   bool   `yaml:"disabled,omitempty"`
	RegionID   int    `yaml:"region_id,omitempty"`
	RegionCode string `yaml:"region_code,omitempty"`
	RegionName string `yaml:"region_name,omitempty"`
}

func (c *Config) Validate() (*Config, error) {
	publicWebUrl, webHost, webPort, err := validatePublicAddr(c.PublicAddr)
	if err != nil {
		return nil, fmt.Errorf("web public addr: %w", err)
	}

	c.PublicUrl = publicWebUrl
	c.derpHost = webHost
	c.derpPort = webPort

	if !c.DERP.Server.Disabled {
		_, stunHost, stunPort, err := validatePublicAddr(c.StunPublicAddr)
		if err != nil {
			return nil, fmt.Errorf("stun public addr: %w", err)
		}

		c.stunHost = stunHost
		c.stunPort = stunPort
	}

	return c, nil
}

func (c *Config) CreateUrl(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	u := url.URL{
		Scheme: c.PublicUrl.Scheme,
		Host:   c.PublicUrl.Host,
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

func (c *Config) DefaultDERPMap() *tailcfg.DERPMap {
	if c.derpHost == c.stunHost {
		return &tailcfg.DERPMap{
			Regions: map[int]*tailcfg.DERPRegion{
				c.DERP.Server.RegionID: {
					RegionID:   c.DERP.Server.RegionID,
					RegionCode: c.DERP.Server.RegionCode,
					RegionName: c.DERP.Server.RegionName,
					Nodes: []*tailcfg.DERPNode{
						{
							RegionID: c.DERP.Server.RegionID,
							Name:     "ionscale",
							HostName: c.derpHost,
							DERPPort: c.derpPort,
							STUNPort: c.stunPort,
						},
					},
				},
			},
		}
	}

	return &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{
			c.DERP.Server.RegionID: {
				RegionID:   c.DERP.Server.RegionID,
				RegionCode: c.DERP.Server.RegionCode,
				RegionName: c.DERP.Server.RegionName,
				Nodes: []*tailcfg.DERPNode{
					{
						RegionID: c.DERP.Server.RegionID,
						Name:     "stun",
						HostName: c.stunHost,
						STUNOnly: true,
						STUNPort: c.stunPort,
					},
					{
						RegionID: c.DERP.Server.RegionID,
						Name:     "derp",
						HostName: c.derpHost,
						DERPPort: c.derpPort,
						STUNPort: -1,
					},
				},
			},
		},
	}
}

// Match ${VAR:default} syntax for variables with default values
var optionalEnvRegex = regexp.MustCompile(`\${([a-zA-Z0-9_]+):([^}]*)}`)

// Match ${VAR} syntax (without default) - these are required
var requiredEnvRegex = regexp.MustCompile(`\${([a-zA-Z0-9_]+)}`)

func expandEnvVars(config []byte) ([]byte, error) {
	var result = config
	var missingVars []string

	result = optionalEnvRegex.ReplaceAllFunc(result, func(match []byte) []byte {
		parts := optionalEnvRegex.FindSubmatch(match)
		envVar := string(parts[1])
		defaultValue := parts[2]

		envValue := os.Getenv(envVar)
		if envValue != "" {
			return []byte(envValue)
		}
		return defaultValue
	})

	result = requiredEnvRegex.ReplaceAllFunc(result, func(match []byte) []byte {
		parts := requiredEnvRegex.FindSubmatch(match)
		envVar := string(parts[1])
		envValue := os.Getenv(envVar)

		if envValue == "" {
			missingVars = append(missingVars, envVar)
			return match
		}

		return []byte(envValue)
	})

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	return result, nil
}
