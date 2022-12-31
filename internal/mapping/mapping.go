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

func ToDNSConfig(m *domain.Machine, tailnet *domain.Tailnet, c *domain.DNSConfig) *tailcfg.DNSConfig {
	certsEnabled := c.HttpsCertsEnabled && config.DNSProviderConfigured()

	sanitizeTailnetName := domain.SanitizeTailnetName(tailnet.Name)
	tailnetDomain := fmt.Sprintf("%s.%s", sanitizeTailnetName, config.MagicDNSSuffix())

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
		routes := make(map[string][]*dnstype.Resolver)

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

	return dnsConfig
}

func ToNode(m *domain.Machine, tailnet *domain.Tailnet, taggedDevicesUser *domain.User, peer bool, connected bool, routeFilter func(m *domain.Machine) []netip.Prefix) (*tailcfg.Node, *tailcfg.UserProfile, error) {
	role := tailnet.IAMPolicy.GetRole(m.User)

	var capabilities []string

	if !peer {
		if !m.HasTags() && role == domain.UserRoleAdmin {
			capabilities = append(capabilities, tailcfg.CapabilityAdmin)
		}

		if tailnet.FileSharingEnabled {
			capabilities = append(capabilities, tailcfg.CapabilityFileSharing)
		}

		if tailnet.SSHEnabled {
			capabilities = append(capabilities, tailcfg.CapabilitySSH)
		}
	}

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
		Services: filterServices(hostinfo.Services),
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

		Hostinfo:     hostInfo.View(),
		Capabilities: capabilities,

		Created: m.CreatedAt.UTC(),

		MachineAuthorized: m.Authorized,
		User:              tailcfg.UserID(m.UserID),
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
		LoginName:   u.Name,
		DisplayName: u.Name,
		Logins:      []tailcfg.LoginID{tailcfg.LoginID(u.ID)},
		Domain:      u.Tailnet.Name,
	}
	login := tailcfg.Login{
		ID:          tailcfg.LoginID(u.ID),
		LoginName:   u.Name,
		DisplayName: u.Name,
		Domain:      u.Tailnet.Name,
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
