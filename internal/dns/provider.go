package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/libdns/azure"
	"github.com/libdns/cloudflare"
	"github.com/libdns/digitalocean"
	"github.com/libdns/googleclouddns"
	"github.com/libdns/libdns"
	"github.com/libdns/route53"
	"strings"
	"time"
)

var factories = map[string]func() libdns.RecordSetter{
	"azure":          azureProvider,
	"cloudflare":     cloudflareProvider,
	"digitalocean":   digitalOceanProvider,
	"googleclouddns": googleCloudDNSProvider,
	"route53":        route53Provider,
}

type Provider interface {
	SetRecord(ctx context.Context, recordType, recordName, value string) error
}

func NewProvider(config config.DNS) (Provider, error) {
	p := config.Provider
	if len(p.Zone) == 0 {
		return nil, nil
	}

	if !strings.HasSuffix(config.MagicDNSSuffix, p.Zone) {
		return nil, fmt.Errorf("invalid MagicDNS suffix [%s], not part of zone [%s]", config.MagicDNSSuffix, p.Zone)
	}

	factory, ok := factories[p.Name]
	if !ok {
		return nil, fmt.Errorf("unknown dns provider: %s", p.Name)
	}

	return newProvider(p.Zone, p.Configuration, factory)
}

func newProvider(zone string, values json.RawMessage, factory func() libdns.RecordSetter) (Provider, error) {
	p := factory()
	if err := json.Unmarshal(values, p); err != nil {
		return nil, err
	}
	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func azureProvider() libdns.RecordSetter {
	return &azure.Provider{}
}

func cloudflareProvider() libdns.RecordSetter {
	return &cloudflare.Provider{}
}

func digitalOceanProvider() libdns.RecordSetter {
	return &digitalocean.Provider{}
}

func googleCloudDNSProvider() libdns.RecordSetter {
	return &googleclouddns.Provider{}
}

func route53Provider() libdns.RecordSetter {
	return &route53.Provider{}
}

type externalProvider struct {
	zone   string
	setter libdns.RecordSetter
}

func (p *externalProvider) SetRecord(ctx context.Context, recordType, recordName, value string) error {
	_, err := p.setter.SetRecords(ctx, p.zone, []libdns.Record{{
		Type:  recordType,
		Name:  libdns.RelativeName(recordName, p.zone),
		Value: value,
		TTL:   1 * time.Minute,
	}})
	return err
}

func fqdn(v string) string {
	if strings.HasSuffix(v, ".") {
		return v
	}
	return fmt.Sprintf("%s.", v)
}
