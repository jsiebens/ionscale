package config

import (
	"fmt"
	"github.com/caddyserver/certmagic"
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
	httpListenAddrKey       = "IONSCALE_HTTP_LISTEN_ADDR"
	httpsListenAddrKey      = "IONSCALE_HTTPS_LISTEN_ADDR"
	serverUrlKey            = "IONSCALE_SERVER_URL"
	keysSystemAdminKeyKey   = "IONSCALE_SYSTEM_ADMIN_KEY"
	keysEncryptionKeyKey    = "IONSCALE_ENCRYPTION_KEY"
	databaseUrlKey          = "IONSCALE_DB_URL"
	tlsDisableKey           = "IONSCALE_TLS_DISABLE"
	tlsCertFileKey          = "IONSCALE_TLS_CERT_FILE"
	tlsKeyFileKey           = "IONSCALE_TLS_KEY_FILE"
	tlsCertMagicCAKey       = "IONSCALE_TLS_CERT_MAGIC_CA"
	tlsCertMagicDomainKey   = "IONSCALE_TLS_CERT_MAGIC_DOMAIN"
	tlsCertMagicEmailKey    = "IONSCALE_TLS_CERT_MAGIC_EMAIL"
	tlsCertMagicStoragePath = "IONSCALE_TLS_CERT_MAGIC_STORAGE_PATH"
	metricsListenAddrKey    = "IONSCALE_METRICS_LISTEN_ADDR"
	loggingLevelKey         = "IONSCALE_LOGGING_LEVEL"
	loggingFormatKey        = "IONSCALE_LOGGING_FORMAT"
	loggingFileKey          = "IONSCALE_LOGGING_FILE"
)

func defaultConfig() *Config {
	return &Config{
		HttpListenAddr:    GetString(httpListenAddrKey, ":8080"),
		HttpsListenAddr:   GetString(httpsListenAddrKey, ":8443"),
		MetricsListenAddr: GetString(metricsListenAddrKey, ":8081"),
		ServerUrl:         GetString(serverUrlKey, "https://localhost:8443"),
		Keys: Keys{
			SystemAdminKey: GetString(keysSystemAdminKeyKey, ""),
			EncryptionKey:  GetString(keysEncryptionKeyKey, ""),
		},
		Database: Database{
			Url: GetString(databaseUrlKey, "ionscale.db"),
		},
		Tls: Tls{
			Disable:              GetBool(tlsDisableKey, false),
			CertFile:             GetString(tlsCertFileKey, ""),
			KeyFile:              GetString(tlsKeyFileKey, ""),
			CertMagicCA:          GetString(tlsCertMagicCAKey, certmagic.LetsEncryptProductionCA),
			CertMagicDomain:      GetString(tlsCertMagicDomainKey, ""),
			CertMagicEmail:       GetString(tlsCertMagicEmailKey, ""),
			CertMagicStoragePath: GetString(tlsCertMagicStoragePath, ""),
		},
		Logging: Logging{
			Level:  GetString(loggingLevelKey, "info"),
			Format: GetString(loggingFormatKey, ""),
			File:   GetString(loggingFileKey, ""),
		},
	}
}

type ServerKeys struct {
	SystemAdminKey key.MachinePrivate
}

type Config struct {
	HttpListenAddr    string   `yaml:"http_listen_addr"`
	HttpsListenAddr   string   `yaml:"https_listen_addr"`
	MetricsListenAddr string   `yaml:"metrics_listen_addr"`
	ServerUrl         string   `yaml:"server_url"`
	Tls               Tls      `yaml:"tls"`
	Logging           Logging  `yaml:"logging"`
	Keys              Keys     `yaml:"keys"`
	Database          Database `yaml:"database"`
}

type Tls struct {
	Disable              bool   `yaml:"disable"`
	CertFile             string `yaml:"cert_file"`
	KeyFile              string `yaml:"key_file"`
	CertMagicDomain      string `yaml:"cert_magic_domain"`
	CertMagicEmail       string `yaml:"cert_magic_email"`
	CertMagicCA          string `yaml:"cert_magic_ca"`
	CertMagicStoragePath string `yaml:"cert_magic_storage_path"`
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
	SystemAdminKey string `yaml:"system_admin_key"`
	EncryptionKey  string `yaml:"encryption_key"`
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

	return &ServerKeys{
		SystemAdminKey: *systemAdminKey,
	}, nil
}
