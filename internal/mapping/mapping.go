package mapping

import (
	"fmt"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"net/netip"
	"slices"
	"strconv"
	"tailscale.com/tailcfg"
	"tailscale.com/types/dnstype"
	"tailscale.com/types/key"
	"time"
)

func ToDNSConfig(m *domain.Machine, tailnet *domain.Tailnet, c *domain.DNSConfig) *tailcfg.DNSConfig {
	certsEnabled := c.HttpsCertsEnabled && config.DNSProviderConfigured()

	sanitizeTailnetName := domain.SanitizeTailnetName(tailnet.Name)
	tailnetDomain := fmt.Sprintf("%s.%s", sanitizeTailnetName, config.MagicDNSSuffix())

	resolvers := make([]*dnstype.Resolver, 0)

	for _, r := range c.Nameservers {
		resolvers = append(resolvers, &dnstype.Resolver{Addr: r})
	}

	dnsConfig := &tailcfg.DNSConfig{}

	var routes = make(map[string][]*dnstype.Resolver)
	var domains []string
	var certDomains []string

	if c.MagicDNS {
		routes[tailnetDomain] = nil
		domains = append(domains, tailnetDomain)
		dnsConfig.Proxied = true

		if certsEnabled {
			certDomains = append(certDomains, fmt.Sprintf("%s.%s", m.CompleteName(), tailnetDomain))
		}
	}

	if c.OverrideLocalDNS {
		dnsConfig.Resolvers = resolvers
	} else {
		dnsConfig.FallbackResolvers = resolvers
	}

	if len(c.Routes) != 0 || certsEnabled {
		for r, s := range c.Routes {
			routeResolver := make([]*dnstype.Resolver, 0)
			for _, addr := range s {
				routeResolver = append(routeResolver, &dnstype.Resolver{Addr: addr})
			}
			routes[r] = routeResolver
		}

		dnsConfig.Routes = routes
	}

	dnsConfig.Domains = append(domains, c.SearchDomains...)
	dnsConfig.CertDomains = certDomains

	dnsConfig.ExitNodeFilteredSet = []string{
		fmt.Sprintf(".%s", config.MagicDNSSuffix()),
	}

	return dnsConfig
}

func ToNode(capVer tailcfg.CapabilityVersion, m *domain.Machine, tailnet *domain.Tailnet, taggedDevicesUser *domain.User, peer bool, connected bool, routeFilter func(m *domain.Machine) []netip.Prefix) (*tailcfg.Node, *tailcfg.UserProfile, error) {
	role := tailnet.IAMPolicy.Get().GetRole(m.User)

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

	if connected {
		allowedIPs = append(allowedIPs, routeFilter(m)...)
	}

	if m.IsAllowedExitNode() {
		allowedIPs = append(allowedIPs, netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0"))
	}

	var derp int
	var legacyDerp string
	if hostinfo.NetInfo != nil {
		derp = hostinfo.NetInfo.PreferredDERP
		legacyDerp = fmt.Sprintf("127.3.3.40:%d", hostinfo.NetInfo.PreferredDERP)
	} else {
		derp = 0
		legacyDerp = "127.3.3.40:0"
	}

	var name = m.CompleteName()

	sanitizedTailnetName := domain.SanitizeTailnetName(m.Tailnet.Name)

	hostInfo := tailcfg.Hostinfo{
		OS:           hostinfo.OS,
		Hostname:     hostinfo.Hostname,
		Services:     filterServices(hostinfo.Services),
		SSH_HostKeys: hostinfo.SSH_HostKeys,
	}

	n := tailcfg.Node{
		ID:               tailcfg.NodeID(m.ID),
		StableID:         tailcfg.StableNodeID(strconv.FormatUint(m.ID, 10)),
		Name:             fmt.Sprintf("%s.%s.%s.", name, sanitizedTailnetName, config.MagicDNSSuffix()),
		Key:              *nKey,
		Machine:          *mKey,
		DiscoKey:         discoKey,
		Addresses:        addrs,
		AllowedIPs:       allowedIPs,
		Endpoints:        endpoints,
		HomeDERP:         derp,
		LegacyDERPString: legacyDerp,

		Hostinfo: hostInfo.View(),
		Created:  m.CreatedAt.UTC(),

		MachineAuthorized: m.Authorized,
		User:              tailcfg.UserID(m.UserID),
	}

	if !peer {
		var capabilities []tailcfg.NodeCapability
		capMap := make(tailcfg.NodeCapMap)

		for _, c := range tailnet.ACLPolicy.Get().NodeCapabilities(m) {
			capabilities = append(capabilities, c)
			capMap[c] = []tailcfg.RawMessage{}
		}

		if !m.HasTags() && role == domain.UserRoleAdmin {
			capabilities = append(capabilities, tailcfg.CapabilityAdmin)
			capMap[tailcfg.CapabilityAdmin] = []tailcfg.RawMessage{}
		}

		if tailnet.FileSharingEnabled {
			capabilities = append(capabilities, tailcfg.CapabilityFileSharing)
			capMap[tailcfg.CapabilityFileSharing] = []tailcfg.RawMessage{}
		}

		if tailnet.SSHEnabled {
			capabilities = append(capabilities, tailcfg.CapabilitySSH)
			capMap[tailcfg.CapabilitySSH] = []tailcfg.RawMessage{}
		}

		if tailnet.DNSConfig.HttpsCertsEnabled {
			capabilities = append(capabilities, tailcfg.CapabilityHTTPS)
			capMap[tailcfg.CapabilityHTTPS] = []tailcfg.RawMessage{}
		}

		// ionscale has no support for Funnel yet, so remove Funnel attribute if set via ACL policy
		{
			slices.DeleteFunc(capabilities, func(c tailcfg.NodeCapability) bool { return c == tailcfg.NodeAttrFunnel })
			delete(capMap, tailcfg.NodeAttrFunnel)
		}

		if capVer >= 74 {
			n.CapMap = capMap
		} else {
			n.Capabilities = capabilities
		}
	}

	if !m.ExpiresAt.IsZero() {
		e := m.ExpiresAt.UTC()
		n.KeyExpiry = e
	}

	if m.KeyExpiryDisabled {
		n.KeyExpiry = time.Time{}
	}

	n.Online = &connected
	if !connected && m.LastSeen != nil {
		n.LastSeen = m.LastSeen
	}

	var user = ToUserProfile(m.User)

	if m.HasTags() {
		n.User = tailcfg.UserID(taggedDevicesUser.ID)
		n.Tags = m.Tags
		user = tailcfg.UserProfile{
			ID:          tailcfg.UserID(taggedDevicesUser.ID),
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

func ToUser(u domain.User) (tailcfg.User, tailcfg.Login) {
	user := tailcfg.User{
		ID:          tailcfg.UserID(u.ID),
		DisplayName: u.Name,
	}
	login := tailcfg.Login{
		ID:          tailcfg.LoginID(u.ID),
		LoginName:   u.Name,
		DisplayName: u.Name,
	}
	return user, login
}

func filterServices(services []tailcfg.Service) []tailcfg.Service {
	result := []tailcfg.Service{}
	for _, s := range services {
		if s.Proto == tailcfg.TCP || s.Proto == tailcfg.UDP {
			continue
		}
		result = append(result, s)
	}
	return result
}
