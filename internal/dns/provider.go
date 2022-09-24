package dns

import (
	"context"
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/libdns/azure"
	"github.com/libdns/cloudflare"
	"github.com/libdns/digitalocean"
	"github.com/libdns/googleclouddns"
	"github.com/libdns/libdns"
	"github.com/libdns/route53"
	"strings"
	"time"
)

type Provider interface {
	SetRecord(ctx context.Context, recordType, recordName, value string) error
}

func NewProvider(config config.DNSProvider) (Provider, error) {
	if len(config.Zone) == 0 {
		return nil, nil
	}

	switch config.Name {
	case "azure":
		return configureProvider(config.Zone, config.Configuration, &azure.Provider{})
	case "cloudflare":
		return configureProvider(config.Zone, config.Configuration, &cloudflare.Provider{})
	case "digitalocean":
		return configureProvider(config.Zone, config.Configuration, &digitalocean.Provider{})
	case "googleclouddns":
		return configureProvider(config.Zone, config.Configuration, &googleclouddns.Provider{})
	case "route53":
		return configureProvider(config.Zone, config.Configuration, &route53.Provider{})
	default:
		return nil, fmt.Errorf("unknown dns provider: %s", config.Name)
	}
}

func configureProvider(zone string, v map[string]string, setter libdns.RecordSetter) (Provider, error) {
	if err := mapping.CopyViaJson(v, setter); err != nil {
		return nil, err
	}
	return &externalProvider{
		zone:   zone,
		setter: setter,
	}, nil
}

type externalProvider struct {
	zone   string
	setter libdns.RecordSetter
}

func (p *externalProvider) SetRecord(ctx context.Context, recordType, recordName, value string) error {
	_, err := p.setter.SetRecords(ctx, fmt.Sprintf("%s.", p.zone), []libdns.Record{{
		Type:  recordType,
		Name:  strings.TrimSuffix(recordName, p.zone),
		Value: value,
		TTL:   1 * time.Minute,
	}})
	return err
}
