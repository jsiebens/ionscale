package config

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/util"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
	"tailscale.com/types/key"
)

func LoadConfig(path string) (*Config, error) {
	config := defaultConfig()

	if len(path) != 0 {
		expandedPath, err := homedir.Expand(path)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadFile(expandedPath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(b, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

const (
	listenAddrKey           = "IONSCALE_LISTEN_ADDR"
	serverUrlKey            = "IONSCALE_SERVER_URL"
	keysSystemAdminKeyKey   = "IONSCALE_SYSTEM_ADMIN_KEY"
	keysControlKeyKey       = "IONSCALE_CONTROL_KEY"
	keysLegacyControlKeyKey = "IONSCALE_LEGACY_CONTROL_KEY"
	databaseUrlKey          = "IONSCALE_DB_URL"
	tlsDisableKey           = "IONSCALE_TLS_DISABLE"
	tlsCertFileKey          = "IONSCALE_TLS_CERT_FILE"
	tlsKeyFileKey           = "IONSCALE_TLS_KEY_FILE"
	metricsListenAddrKey    = "IONSCALE_METRICS_LISTEN_ADDR"
	loggingLevelKey         = "IONSCALE_LOGGING_LEVEL"
	loggingFormatKey        = "IONSCALE_LOGGING_FORMAT"
	loggingFileKey          = "IONSCALE_LOGGING_FILE"
)

func defaultConfig() *Config {
	return &Config{
		ListenAddr: GetString(listenAddrKey, ":8000"),
		ServerUrl:  GetString(serverUrlKey, "https://localhost:8000"),
		Keys: Keys{
			SystemAdminKey:   GetString(keysSystemAdminKeyKey, ""),
			ControlKey:       GetString(keysControlKeyKey, ""),
			LegacyControlKey: GetString(keysLegacyControlKeyKey, ""),
		},
		Database: Database{
			Url: GetString(databaseUrlKey, "ionscale.db"),
		},
		Tls: Tls{
			Disable:  GetBool(tlsDisableKey, false),
			CertFile: GetString(tlsCertFileKey, ""),
			KeyFile:  GetString(tlsKeyFileKey, ""),
		},
		Metrics: Metrics{ListenAddr: GetString(metricsListenAddrKey, ":8001")},
		Logging: Logging{
			Level:  GetString(loggingLevelKey, "info"),
			Format: GetString(loggingFormatKey, ""),
			File:   GetString(loggingFileKey, ""),
		},
	}
}

type ServerKeys struct {
	SystemAdminKey   key.MachinePrivate
	ControlKey       key.MachinePrivate
	LegacyControlKey key.MachinePrivate
}

type Config struct {
	ListenAddr string   `yaml:"listen_addr"`
	ServerUrl  string   `yaml:"server_url"`
	Tls        Tls      `yaml:"tls"`
	Metrics    Metrics  `yaml:"metrics"`
	Logging    Logging  `yaml:"logging"`
	Keys       Keys     `yaml:"keys"`
	Database   Database `yaml:"database"`
}

type Metrics struct {
	ListenAddr string `yaml:"listen_addr"`
}

type Tls struct {
	Disable  bool   `yaml:"disable"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type Logging struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

type Database struct {
	Url string `yaml:"url"`
}

type Keys struct {
	SystemAdminKey   string `yaml:"system_admin_key"`
	ControlKey       string `yaml:"control_key"`
	LegacyControlKey string `yaml:"legacy_control_key"`
}

func (c *Config) CreateUrl(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	return strings.TrimSuffix(c.ServerUrl, "/") + "/" + strings.TrimPrefix(path, "/")
}

func (c *Config) ReadServerKeys() (*ServerKeys, error) {
	systemAdminKey, err := util.ParseMachinePrivateKey(c.Keys.SystemAdminKey)
	if err != nil {
		return nil, fmt.Errorf("error reading system admin key: %v", err)
	}

	controlKey, err := util.ParseMachinePrivateKey(c.Keys.ControlKey)
	if err != nil {
		return nil, fmt.Errorf("error reading control key: %v", err)
	}

	legacyControlKey, err := util.ParseMachinePrivateKey(c.Keys.LegacyControlKey)
	if err != nil {
		return nil, fmt.Errorf("error reading legacy control key: %v", err)
	}

	return &ServerKeys{
		SystemAdminKey:   *systemAdminKey,
		ControlKey:       *controlKey,
		LegacyControlKey: *legacyControlKey,
	}, nil
}
