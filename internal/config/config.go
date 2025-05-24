package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mitchellh/go-homedir"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/yaml"
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

	keepAliveInterval = time.Duration(cfg.PollNet.KeepAliveInterval)
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
			KeepAliveInterval: Duration(defaultKeepAliveInterval),
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
	ListenAddr        string   `json:"listen_addr,omitempty"`
	StunListenAddr    string   `json:"stun_listen_addr,omitempty"`
	MetricsListenAddr string   `json:"metrics_listen_addr,omitempty"`
	PublicAddr        string   `json:"public_addr,omitempty"`
	StunPublicAddr    string   `json:"stun_public_addr,omitempty"`
	Tls               Tls      `json:"tls,omitempty"`
	PollNet           PollNet  `json:"poll_net,omitempty"`
	Keys              Keys     `json:"keys,omitempty"`
	Database          Database `json:"database,omitempty"`
	Auth              Auth     `json:"auth,omitempty"`
	DNS               DNS      `json:"dns,omitempty"`
	DERP              DERP     `json:"derp,omitempty"`
	Logging           Logging  `json:"logging,omitempty"`

	PublicUrl *url.URL `json:"-"`

	stunHost string
	stunPort int
	derpHost string
	derpPort int
}

type Tls struct {
	Disable     bool   `json:"disable"`
	ForceHttps  bool   `json:"force_https"`
	CertFile    string `json:"cert_file,omitempty"`
	KeyFile     string `json:"key_file,omitempty"`
	AcmeEnabled bool   `json:"acme,omitempty"`
	AcmeEmail   string `json:"acme_email,omitempty"`
	AcmeCA      string `json:"acme_ca,omitempty"`
}

type PollNet struct {
	KeepAliveInterval Duration `json:"keep_alive_interval"`
}

type Logging struct {
	Level  string `json:"level,omitempty"`
	Format string `json:"format,omitempty"`
	File   string `json:"file,omitempty"`
}

type Database struct {
	Type            string   `json:"type,omitempty"`
	Url             string   `json:"url,omitempty"`
	MaxOpenConns    int      `json:"max_open_conns,omitempty"`
	MaxIdleConns    int      `json:"max_idle_conns,omitempty"`
	ConnMaxLifetime Duration `json:"conn_max_life_time,omitempty"`
	ConnMaxIdleTime Duration `json:"conn_max_idle_time,omitempty"`
}

type Keys struct {
	ControlKey       string `json:"control_key,omitempty"`
	LegacyControlKey string `json:"legacy_control_key,omitempty"`
	SystemAdminKey   string `json:"system_admin_key,omitempty"`
}

type Auth struct {
	Provider          AuthProvider      `json:"provider,omitempty"`
	SystemAdminPolicy SystemAdminPolicy `json:"system_admins"`
}

type AuthProvider struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"additional_scopes" `
}

type DNS struct {
	MagicDNSSuffix string      `json:"magic_dns_suffix"`
	Provider       DNSProvider `json:"provider,omitempty"`
}

type DNSProvider struct {
	Name          string          `json:"name"`
	PluginPath    string          `json:"plugin_path"`
	Zone          string          `json:"zone"`
	Configuration json.RawMessage `json:"config"`
}

type SystemAdminPolicy struct {
	Subs    []string `json:"subs,omitempty"`
	Emails  []string `json:"emails,omitempty"`
	Filters []string `json:"filters,omitempty"`
}

type DERP struct {
	Server  DERPServer `json:"server,omitempty"`
	Sources []string   `json:"sources,omitempty"`
}

type DERPServer struct {
	Disabled   bool   `json:"disabled,omitempty"`
	RegionID   int    `json:"region_id,omitempty"`
	RegionCode string `json:"region_code,omitempty"`
	RegionName string `json:"region_name,omitempty"`
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

type Duration time.Duration

func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(value)
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return fmt.Errorf("invalid duration")
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
