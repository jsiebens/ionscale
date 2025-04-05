package dns

import (
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/pkg/sdk/dnsplugin"
	"github.com/libdns/azure"
	"github.com/libdns/cloudflare"
	"go.uber.org/zap"
	"strings"
)

func NewProvider(config config.DNS) (string, dnsplugin.Provider, error) {
	p := config.Provider
	if len(p.Zone) == 0 {
		return "", nil, nil
	}

	if !strings.HasSuffix(config.MagicDNSSuffix, p.Zone) {
		return "", nil, fmt.Errorf("invalid MagicDNS suffix [%s], not part of zone [%s]", config.MagicDNSSuffix, p.Zone)
	}

	pluginInfo := dnsplugin.Get(p.Name)

	// fallback to builtin plugins
	if pluginInfo == nil {
		pluginInfo = builtInPlugins[p.Name]

	}

	if pluginInfo == nil {
		return "", nil, fmt.Errorf("unknown dns provider: %s", p.Name)
	}

	provider, err := newProvider(p.Configuration, pluginInfo)
	return fqdn(p.Zone), provider, err
}

func newProvider(values json.RawMessage, plugin *dnsplugin.PluginInfo) (dnsplugin.Provider, error) {
	p := plugin.New()
	if err := json.Unmarshal(values, p); err != nil {
		return nil, err
	}
	return p, nil
}

func fqdn(v string) string {
	if strings.HasSuffix(v, ".") {
		return v
	}
	return fmt.Sprintf("%s.", v)
}

var builtInPlugins = map[string]*dnsplugin.PluginInfo{
	"azure":      {New: azureProvider},
	"cloudflare": {New: cloudflareProvider},

	/*
		"digitalocean":   {New: digitalOceanProvider},
		"googleclouddns": {New: googleCloudDNSProvider},
		"route53":        {New: route53Provider},
	*/
}

func cloudflareProvider() dnsplugin.Provider {
	zap.L().Warn("Builtin cloudflare DNS plugin is deprecated and will be removed in a future release.")
	return &cloudflare.Provider{}
}

func azureProvider() dnsplugin.Provider {
	zap.L().Warn("Builtin azure DNS plugin is deprecated and will be removed in a future release.")
	return &azure.Provider{}
}

/*
func digitalOceanProvider() dnsplugin.Provider {
	zap.L().Warn("Builtin digitalocean DNS plugin is deprecated and will be removed in a future release.")
	return &digitalocean.Provider{}
}

func googleCloudDNSProvider() dnsplugin.Provider {
	zap.L().Warn("Builtin googleclouddns DNS plugin is deprecated and will be removed in a future release.")
	return &googleclouddns.Provider{}
}

func route53Provider() dnsplugin.Provider {
	zap.L().Warn("Builtin route53 DNS plugin is deprecated and will be removed in a future release.")
	return &route53.Provider{}
}
*/
