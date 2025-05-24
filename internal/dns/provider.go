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
	"go.uber.org/zap"
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

	if p.Name == "" && p.PluginPath == "" {
		return nil, fmt.Errorf("invalid dns provider configuration, either name or plugin_path should be set")
	}

	if p.Name != "" && p.PluginPath != "" {
		return nil, fmt.Errorf("invalid dns provider configuration, only one of name or plugin_path should be set")
	}

	if !strings.HasSuffix(config.MagicDNSSuffix, p.Zone) {
		return nil, fmt.Errorf("invalid MagicDNS suffix [%s], not part of zone [%s]", config.MagicDNSSuffix, p.Zone)
	}

	factory, ok := factories[p.Name]
	if ok {
		return newProvider(p.Zone, p.Configuration, factory)
	}

	return newPluginManager(p.PluginPath, fqdn(p.Zone), p.Configuration)
}

func newProvider(zone string, values json.RawMessage, factory func() libdns.RecordSetter) (Provider, error) {
	p := factory()
	if err := json.Unmarshal(values, p); err != nil {
		return nil, err
	}
	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func azureProvider() libdns.RecordSetter {
	zap.L().Warn("Builtin azure DNS plugin is deprecated and will be removed in a future release.")
	return &azure.Provider{}
}

func cloudflareProvider() libdns.RecordSetter {
	zap.L().Warn("Builtin cloudflare DNS plugin is deprecated and will be removed in a future release.")
	return &cloudflare.Provider{}
}

func digitalOceanProvider() libdns.RecordSetter {
	zap.L().Warn("Builtin digitalocean DNS plugin is deprecated and will be removed in a future release.")
	return &digitalocean.Provider{}
}

func googleCloudDNSProvider() libdns.RecordSetter {
	zap.L().Warn("Builtin googleclouddns DNS plugin is deprecated and will be removed in a future release.")
	return &googleclouddns.Provider{}
}

func route53Provider() libdns.RecordSetter {
	zap.L().Warn("Builtin route53 DNS plugin is deprecated and will be removed in a future release.")
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
