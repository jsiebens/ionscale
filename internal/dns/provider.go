package dns

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/libdns/cloudflare"
	"github.com/libdns/googleclouddns"
	"github.com/libdns/libdns"
	"github.com/libdns/route53"
)

type Provider interface {
	libdns.RecordSetter
}

func NewProvider(config config.DNSProvider) (Provider, error) {
	if len(config.Zone) == 0 {
		return nil, nil
	}

	mapping.CertDNSSuffix = config.Zone

	switch config.Name {
	case "cloudflare":
		return configureProvider(config.Configuration, &cloudflare.Provider{})
	case "googleclouddns":
		return configureProvider(config.Configuration, &googleclouddns.Provider{})
	case "route53":
		return configureProvider(config.Configuration, &route53.Provider{})
	default:
		return nil, fmt.Errorf("Unknown provider: %s", config.Name)
	}
}

func configureProvider(v map[string]string, provider Provider) (Provider, error) {
	if err := mapping.CopyViaJson(v, provider); err != nil {
		return nil, err
	}
	return provider, nil
}
