package mapping

import (
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"net/netip"
	"strconv"
	"tailscale.com/tailcfg"
	"tailscale.com/types/dnstype"
	"tailscale.com/types/key"
	"time"
)

func CopyViaJson[F any, T any](f F, t T) error {
	raw, err := json.Marshal(f)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, t); err != nil {
		return err
	}

	return nil
}

func ToDNSConfig(m *domain.Machine, peers []domain.Machine, tailnet *domain.Tailnet, c *domain.DNSConfig) *tailcfg.DNSConfig {
	certDNSSuffix := config.CertDNSSuffix()
	certsEnabled := c.HttpsCertsEnabled && len(certDNSSuffix) != 0

	tailnetDomain := domain.SanitizeTailnetName(tailnet.Name)

	var certDomain = ""
	if certsEnabled {
		certDomain = domain.SanitizeTailnetName(*tailnet.Alias)
	}

	resolvers := []*dnstype.Resolver{}
	for _, r := range c.Nameservers {
		resolver := &dnstype.Resolver{
			Addr: r,
		}
		resolvers = append(resolvers, resolver)
	}

	dnsConfig := &tailcfg.DNSConfig{}

	var domains []string
	var certDomains []string

	if c.MagicDNS {
		domains = append(domains, fmt.Sprintf("%s.%s", tailnetDomain, config.MagicDNSSuffix()))
		dnsConfig.Proxied = true

		if certsEnabled {
			domains = append(domains, fmt.Sprintf("%s.%s", certDomain, certDNSSuffix))
			certDomains = append(certDomains, fmt.Sprintf("%s.%s.%s", m.CompleteName(), certDomain, certDNSSuffix))
		}
	}

	if c.OverrideLocalDNS {
		dnsConfig.Resolvers = resolvers
	} else {
		dnsConfig.FallbackResolvers = resolvers
	}

	if len(c.Routes) != 0 || certsEnabled {
		routes := make(map[string][]*dnstype.Resolver)

		if certsEnabled {
			routes[fmt.Sprintf("%s.", certDNSSuffix)] = nil
		}

		for r, s := range c.Routes {
			routeResolver := []*dnstype.Resolver{}
			for _, addr := range s {
				resolver := &dnstype.Resolver{Addr: addr}
				routeResolver = append(routeResolver, resolver)
			}
			routes[r] = routeResolver
			domains = append(domains, r)
		}
		dnsConfig.Routes = routes
	}

	dnsConfig.Domains = domains
	dnsConfig.CertDomains = certDomains

	if certsEnabled {
		var extraRecords = []tailcfg.DNSRecord{{
			Name:  fmt.Sprintf("%s.%s.%s", m.CompleteName(), certDomain, certDNSSuffix),
			Value: m.IPv4.String(),
		}}

		for _, p := range peers {
			extraRecords = append(extraRecords, tailcfg.DNSRecord{
				Name:  fmt.Sprintf("%s.%s.%s", p.CompleteName(), certDomain, certDNSSuffix),
				Value: p.IPv4.String(),
			})
		}

		dnsConfig.ExtraRecords = extraRecords
	}

	return dnsConfig
}

func ToNode(m *domain.Machine) (*tailcfg.Node, *tailcfg.UserProfile, error) {
	nKey, err := util.ParseNodePublicKey(m.NodeKey)
	if err != nil {
		return nil, nil, err
	}

	mKey, err := util.ParseMachinePublicKey(m.MachineKey)
	if err != nil {
		return nil, nil, err
	}

	var discoKey key.DiscoPublic
	if m.DiscoKey != "" {
		dKey, err := util.ParseDiscoPublicKey(m.DiscoKey)
		if err != nil {
			return nil, nil, err
		}
		discoKey = *dKey
	}

	endpoints := m.Endpoints
	hostinfo := tailcfg.Hostinfo(m.HostInfo)

	var addrs []netip.Prefix
	var allowedIPs []netip.Prefix

	if m.IPv4.IsValid() {
		ipv4, err := m.IPv4.Prefix(32)
		if err != nil {
			return nil, nil, err
		}
		addrs = append(addrs, ipv4)
		allowedIPs = append(allowedIPs, ipv4)
	}

	if m.IPv6.IsValid() {
		ipv6, err := m.IPv6.Prefix(128)
		if err != nil {
			return nil, nil, err
		}
		addrs = append(addrs, ipv6)
		allowedIPs = append(allowedIPs, ipv6)
	}

	allowedIPs = append(allowedIPs, m.AllowIPs...)
	allowedIPs = append(allowedIPs, m.AutoAllowIPs...)

	var derp string
	if hostinfo.NetInfo != nil {
		derp = fmt.Sprintf("127.3.3.40:%d", hostinfo.NetInfo.PreferredDERP)
	} else {
		derp = "127.3.3.40:0"
	}

	var name = m.CompleteName()

	sanitizedTailnetName := domain.SanitizeTailnetName(m.Tailnet.Name)

	hostInfo := tailcfg.Hostinfo{
		OS:       hostinfo.OS,
		Hostname: hostinfo.Hostname,
		Services: hostinfo.Services,
	}

	n := tailcfg.Node{
		ID:         tailcfg.NodeID(m.ID),
		StableID:   tailcfg.StableNodeID(strconv.FormatUint(m.ID, 10)),
		Name:       fmt.Sprintf("%s.%s.%s.", name, sanitizedTailnetName, config.MagicDNSSuffix()),
		Key:        *nKey,
		Machine:    *mKey,
		DiscoKey:   discoKey,
		Addresses:  addrs,
		AllowedIPs: allowedIPs,
		Endpoints:  endpoints,
		DERP:       derp,

		Hostinfo: hostInfo.View(),

		Created: m.CreatedAt.UTC(),

		MachineAuthorized: true,
		User:              tailcfg.UserID(m.UserID),
	}

	if !m.ExpiresAt.IsZero() {
		e := m.ExpiresAt.UTC()
		n.KeyExpiry = e
	}

	if m.KeyExpiryDisabled {
		n.KeyExpiry = time.Time{}
	}

	if m.LastSeen != nil {
		l := m.LastSeen.UTC()
		online := m.LastSeen.After(time.Now().Add(-config.KeepAliveInterval()))
		n.LastSeen = &l
		n.Online = &online
	}

	var user = ToUserProfile(m.User)

	if m.HasTags() {
		n.User = tailcfg.UserID(m.ID)
		user = tailcfg.UserProfile{
			ID:          tailcfg.UserID(m.ID),
			LoginName:   "tagged-devices",
			DisplayName: "Tagged Devices",
		}
	}

	return &n, &user, nil
}

func ToUserProfile(u domain.User) tailcfg.UserProfile {
	profile := tailcfg.UserProfile{
		ID:          tailcfg.UserID(u.ID),
		LoginName:   u.Name,
		DisplayName: u.Name,
	}
	return profile
}

func ToUserProfiles(users domain.Users) []tailcfg.UserProfile {
	var profiles []tailcfg.UserProfile
	for _, u := range users {
		profiles = append(profiles, ToUserProfile(u))
	}
	return profiles
}
