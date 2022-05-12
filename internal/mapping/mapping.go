package mapping

import (
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/util"
	"inet.af/netaddr"
	"strconv"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"tailscale.com/util/dnsname"
)

const NetworkMagicDNSSuffix = "ionscale.net"

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

func ToNode(m *domain.Machine, connected bool) (*tailcfg.Node, error) {
	nKey, err := util.ParseNodePublicKey(m.NodeKey)
	if err != nil {
		return nil, err
	}

	mKey, err := util.ParseMachinePublicKey(m.MachineKey)
	if err != nil {
		return nil, err
	}

	var discoKey key.DiscoPublic
	if m.DiscoKey != "" {
		dKey, err := util.ParseDiscoPublicKey(m.DiscoKey)
		if err != nil {
			return nil, err
		}
		discoKey = *dKey
	}

	endpoints := m.Endpoints
	hostinfo := tailcfg.Hostinfo(m.HostInfo)

	var addrs []netaddr.IPPrefix
	var allowedIPs []netaddr.IPPrefix

	if !m.IPv4.IsZero() {
		ipv4, err := m.IPv4.Prefix(32)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, ipv4)
		allowedIPs = append(allowedIPs, ipv4)
	}

	if !m.IPv6.IsZero() {
		ipv6, err := m.IPv6.Prefix(128)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, ipv6)
		allowedIPs = append(allowedIPs, ipv6)
	}

	var derp string
	if hostinfo.NetInfo != nil {
		derp = fmt.Sprintf("127.3.3.40:%d", hostinfo.NetInfo.PreferredDERP)
	} else {
		derp = "127.3.3.40:0"
	}

	var name = m.Name
	if m.NameIdx != 0 {
		name = fmt.Sprintf("%s-%d", m.Name, m.NameIdx)
	}

	sanitizedTailnetName := dnsname.SanitizeHostname(m.Tailnet.Name)

	hostInfo := tailcfg.Hostinfo{
		OS:       hostinfo.OS,
		Hostname: hostinfo.Hostname,
		Services: hostinfo.Services,
	}

	n := tailcfg.Node{
		ID:         tailcfg.NodeID(m.ID),
		StableID:   tailcfg.StableNodeID(strconv.FormatUint(m.ID, 10)),
		Name:       fmt.Sprintf("%s.%s.%s.", name, sanitizedTailnetName, NetworkMagicDNSSuffix),
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

	if m.ExpiresAt != nil {
		e := m.ExpiresAt.UTC()
		n.KeyExpiry = e
	}

	n.Online = &connected
	if !connected && m.LastSeen != nil {
		l := m.LastSeen.UTC()
		n.LastSeen = &l
	}

	return &n, nil
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
