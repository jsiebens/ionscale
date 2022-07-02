package config

import (
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/jsiebens/ionscale/internal/key"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
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
		Provider: Provider{},
		Logging: Logging{
			Level:  GetString(loggingLevelKey, "info"),
			Format: GetString(loggingFormatKey, ""),
			File:   GetString(loggingFileKey, ""),
		},
	}
}

type ServerKeys struct {
	SystemAdminKey key.ServerPrivate
}

type Config struct {
	HttpListenAddr    string   `yaml:"http_listen_addr,omitempty"`
	HttpsListenAddr   string   `yaml:"https_listen_addr,omitempty"`
	MetricsListenAddr string   `yaml:"metrics_listen_addr,omitempty"`
	ServerUrl         string   `yaml:"server_url,omitempty"`
	Tls               Tls      `yaml:"tls,omitempty"`
	Logging           Logging  `yaml:"logging,omitempty"`
	Keys              Keys     `yaml:"keys,omitempty"`
	Database          Database `yaml:"database,omitempty"`
	Provider          Provider `yaml:"oidc,omitempty"`
}

type Tls struct {
	Disable              bool   `yaml:"disable"`
	CertFile             string `yaml:"cert_file,omitempty"`
	KeyFile              string `yaml:"key_file,omitempty"`
	CertMagicDomain      string `yaml:"cert_magic_domain,omitempty"`
	CertMagicEmail       string `yaml:"cert_magic_email,omitempty"`
	CertMagicCA          string `yaml:"cert_magic_ca,omitempty"`
	CertMagicStoragePath string `yaml:"cert_magic_storage_path,omitempty"`
}

type Logging struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`
	File   string `yaml:"file,omitempty"`
}

type Database struct {
	Url string `yaml:"url,omitempty"`
}

type Keys struct {
	SystemAdminKey string `yaml:"system_admin_key,omitempty"`
}

type Provider struct {
	Issuer       string   `yaml:"issuer"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Scopes       []string `yaml:"additional_scopes"`
}

func (c *Config) CreateUrl(format string, a ...interface{}) string {
	path := fmt.Sprintf(format, a...)
	return strings.TrimSuffix(c.ServerUrl, "/") + "/" + strings.TrimPrefix(path, "/")
}

func (c *Config) ReadServerKeys() (*ServerKeys, error) {
	systemAdminKey, err := key.ParsePrivateKey(c.Keys.SystemAdminKey)
	if err != nil {
		return nil, fmt.Errorf("error reading system admin key: %v", err)
	}

	return &ServerKeys{
		SystemAdminKey: *systemAdminKey,
	}, nil
}
