package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/netip"
	"strconv"
	"strings"
	"tailscale.com/tailcfg"
)

type ACLPolicy struct {
	Groups    map[string][]string `json:"groups,omitempty"`
	Hosts     map[string]string   `json:"hosts,omitempty"`
	ACLs      []ACL               `json:"acls"`
	TagOwners map[string][]string `json:"tag_owners"`
}

type ACL struct {
	Action string   `json:"action"`
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
}

func DefaultPolicy() ACLPolicy {
	return ACLPolicy{
		ACLs: []ACL{
			{
				Action: "accept",
				Src:    []string{"*"},
				Dst:    []string{"*:*"},
			},
		},
	}
}

func (a ACLPolicy) CheckTags(tags []string) error {
	var result *multierror.Error
	for _, t := range tags {
		if _, ok := a.TagOwners[t]; !ok {
			result = multierror.Append(result, fmt.Errorf("tag [%s] is invalid or not permitted", t))
		}
	}
	return result.ErrorOrNil()
}

func (a ACLPolicy) CheckTagOwners(tags []string, p *User) error {
	var result *multierror.Error
	for _, t := range tags {
		if ok := a.IsTagOwner(t, p); !ok {
			result = multierror.Append(result, fmt.Errorf("tag [%s] is invalid or not permitted", t))
		}
	}
	return result.ErrorOrNil()
}

func (a ACLPolicy) IsTagOwner(tag string, p *User) bool {
	if p.UserType == UserTypeService {
		return true
	}
	if tagOwners, ok := a.TagOwners[tag]; ok {
		return a.validateTagOwners(tagOwners, p)
	}
	return false
}

func (a ACLPolicy) validateTagOwners(tagOwners []string, p *User) bool {
	for _, alias := range tagOwners {
		if strings.HasPrefix(alias, "group:") {
			if group, ok := a.Groups[alias]; ok {
				for _, groupMember := range group {
					if groupMember == p.Name {
						return true
					}
				}
			}
		} else {
			if alias == p.Name {
				return true
			}
		}
	}
	return false
}

func (a ACLPolicy) IsValidPeer(src *Machine, dest *Machine) bool {
	if !src.HasTags() && !dest.HasTags() && dest.HasUser(src.User.Name) {
		return true
	}

	for _, acl := range a.ACLs {
		allDestPorts := a.expandMachineToDstPorts(dest, acl.Dst)
		if len(allDestPorts) == 0 {
			continue
		}

		for _, alias := range acl.Src {
			if len(a.expandMachineAlias(src, alias, true)) != 0 {
				return true
			}
		}
	}
	return false
}

func (a ACLPolicy) BuildFilterRules(srcs []Machine, dst *Machine) []tailcfg.FilterRule {
	var rules []tailcfg.FilterRule

	for _, acl := range a.ACLs {
		allDestPorts := a.expandMachineToDstPorts(dst, acl.Dst)
		if len(allDestPorts) == 0 {
			continue
		}

		var allSrcIPsSet = &StringSet{}
		for _, alias := range acl.Src {
			for _, src := range srcs {
				srcIPs := a.expandMachineAlias(&src, alias, true)
				allSrcIPsSet.Add(srcIPs...)
			}
		}

		allSrcIPs := allSrcIPsSet.Items()

		if len(allSrcIPs) == 0 {
			allSrcIPs = nil
		}

		rule := tailcfg.FilterRule{
			SrcIPs:   allSrcIPs,
			DstPorts: allDestPorts,
		}

		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return []tailcfg.FilterRule{{}}
	}

	return rules
}

func (a ACLPolicy) expandMachineToDstPorts(m *Machine, ports []string) []tailcfg.NetPortRange {
	allDestRanges := []tailcfg.NetPortRange{}
	for _, d := range ports {
		ranges := a.expandMachineDestToNetPortRanges(m, d)
		allDestRanges = append(allDestRanges, ranges...)
	}
	return allDestRanges
}

func (a ACLPolicy) expandMachineDestToNetPortRanges(m *Machine, dest string) []tailcfg.NetPortRange {
	tokens := strings.Split(dest, ":")
	if len(tokens) < 2 || len(tokens) > 3 {
		return nil
	}

	var alias string
	if len(tokens) == 2 {
		alias = tokens[0]
	} else {
		alias = fmt.Sprintf("%s:%s", tokens[0], tokens[1])
	}

	ports, err := a.expandValuePortToPortRange(tokens[len(tokens)-1])
	if err != nil {
		return nil
	}

	ips := a.expandMachineAlias(m, alias, false)
	if len(ips) == 0 {
		return nil
	}

	dests := []tailcfg.NetPortRange{}
	for _, d := range ips {
		for _, p := range ports {
			pr := tailcfg.NetPortRange{
				IP:    d,
				Ports: p,
			}
			dests = append(dests, pr)
		}
	}

	return dests
}

func (a ACLPolicy) expandMachineAlias(m *Machine, alias string, src bool) []string {
	if alias == "*" {
		if alias == "*" {
			return []string{"*"}
		}
	}

	if strings.Contains(alias, "@") && !m.HasTags() && m.HasUser(alias) {
		return []string{m.IPv4.String(), m.IPv6.String()}
	}

	if strings.HasPrefix(alias, "group:") && !m.HasTags() {
		users, ok := a.Groups[alias]

		if !ok {
			return []string{}
		}

		for _, u := range users {
			if m.HasUser(u) {
				return []string{m.IPv4.String(), m.IPv6.String()}
			}
		}

		return []string{}
	}

	if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
		return []string{m.IPv4.String(), m.IPv6.String()}
	}

	if h, ok := a.Hosts[alias]; ok {
		alias = h
	}

	if src {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.HasIP(ip) {
			return []string{ip.String()}
		}
	} else {
		ip, err := netip.ParseAddr(alias)
		if err == nil && m.IsAllowedIP(ip) {
			return []string{ip.String()}
		}

		prefix, err := netip.ParsePrefix(alias)
		if err == nil && m.IsAllowedIPPrefix(prefix) {
			return []string{prefix.String()}
		}
	}

	return []string{}
}

func (a ACLPolicy) expandValuePortToPortRange(s string) ([]tailcfg.PortRange, error) {
	if s == "*" {
		return []tailcfg.PortRange{{First: 0, Last: 65535}}, nil
	}

	ports := []tailcfg.PortRange{}
	for _, p := range strings.Split(s, ",") {
		rang := strings.Split(p, "-")
		if len(rang) == 1 {
			pi, err := strconv.ParseUint(rang[0], 10, 16)
			if err != nil {
				return nil, err
			}
			ports = append(ports, tailcfg.PortRange{
				First: uint16(pi),
				Last:  uint16(pi),
			})
		} else if len(rang) == 2 {
			start, err := strconv.ParseUint(rang[0], 10, 16)
			if err != nil {
				return nil, err
			}
			last, err := strconv.ParseUint(rang[1], 10, 16)
			if err != nil {
				return nil, err
			}
			ports = append(ports, tailcfg.PortRange{
				First: uint16(start),
				Last:  uint16(last),
			})
		} else {
			return nil, fmt.Errorf("invalid format")
		}
	}
	return ports, nil
}

func (i *ACLPolicy) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, i)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i ACLPolicy) Value() (driver.Value, error) {
	bytes, err := json.Marshal(i)
	return bytes, err
}

// GormDataType gorm common data type
func (ACLPolicy) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (ACLPolicy) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

type StringSet struct {
	items map[string]bool
}

func (s *StringSet) Add(t ...string) *StringSet {
	if s.items == nil {
		s.items = make(map[string]bool)
	}

	for _, v := range t {
		s.items[v] = true
	}

	return s
}

func (s *StringSet) Items() []string {
	items := []string{}
	for i := range s.items {
		items = append(items, i)
	}
	return items
}
