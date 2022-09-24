package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"tailscale.com/tailcfg"
)

const (
	AutoGroupSelf    = "autogroup:self"
	AutoGroupMembers = "autogroup:members"
)

type ACLPolicy struct {
	Groups    map[string][]string `json:"groups,omitempty"`
	Hosts     map[string]string   `json:"hosts,omitempty"`
	ACLs      []ACL               `json:"acls"`
	TagOwners map[string][]string `json:"tagowners"`
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
		selfDestPorts, allDestPorts := a.expandMachineToDstPorts(dest, acl.Dst)
		if len(selfDestPorts) != 0 {
			for _, alias := range acl.Src {
				if len(a.expandMachineAlias(src, alias, true, &dest.User)) != 0 {
					return true
				}
			}
		}
		if len(allDestPorts) != 0 {
			for _, alias := range acl.Src {
				if len(a.expandMachineAlias(src, alias, true, nil)) != 0 {
					return true
				}
			}
		}
	}

	return false
}

func (a ACLPolicy) BuildFilterRules(srcs []Machine, dst *Machine) []tailcfg.FilterRule {
	var rules []tailcfg.FilterRule

	transform := func(src []string, destPorts []tailcfg.NetPortRange, u *User) tailcfg.FilterRule {
		var allSrcIPsSet = &StringSet{}
		for _, alias := range src {
			for _, src := range srcs {
				srcIPs := a.expandMachineAlias(&src, alias, true, u)
				allSrcIPsSet.Add(srcIPs...)
			}
		}

		allSrcIPs := allSrcIPsSet.Items()

		if len(allSrcIPs) == 0 {
			allSrcIPs = nil
		}

		return tailcfg.FilterRule{
			SrcIPs:   allSrcIPs,
			DstPorts: destPorts,
		}
	}

	for _, acl := range a.ACLs {
		selfDestPorts, allDestPorts := a.expandMachineToDstPorts(dst, acl.Dst)
		if len(selfDestPorts) != 0 {
			rules = append(rules, transform(acl.Src, selfDestPorts, &dst.User))
		}
		if len(allDestPorts) != 0 {
			rules = append(rules, transform(acl.Src, allDestPorts, nil))
		}
	}

	if len(rules) == 0 {
		return []tailcfg.FilterRule{{}}
	}

	return rules
}

func (a ACLPolicy) expandMachineToDstPorts(m *Machine, ports []string) ([]tailcfg.NetPortRange, []tailcfg.NetPortRange) {
	selfDestRanges := []tailcfg.NetPortRange{}
	otherDestRanges := []tailcfg.NetPortRange{}
	for _, d := range ports {
		self, ranges := a.expandMachineDestToNetPortRanges(m, d)
		if self {
			selfDestRanges = append(selfDestRanges, ranges...)
		} else {
			otherDestRanges = append(otherDestRanges, ranges...)
		}
	}
	return selfDestRanges, otherDestRanges
}

func (a ACLPolicy) expandMachineDestToNetPortRanges(m *Machine, dest string) (bool, []tailcfg.NetPortRange) {
	tokens := strings.Split(dest, ":")
	if len(tokens) < 2 || len(tokens) > 3 {
		return false, nil
	}

	var alias string
	if len(tokens) == 2 {
		alias = tokens[0]
	} else {
		alias = fmt.Sprintf("%s:%s", tokens[0], tokens[1])
	}

	ports, err := a.expandValuePortToPortRange(tokens[len(tokens)-1])
	if err != nil {
		return false, nil
	}

	ips := a.expandMachineAlias(m, alias, false, nil)
	if len(ips) == 0 {
		return false, nil
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

	return alias == AutoGroupSelf, dests
}

func (a ACLPolicy) expandMachineAlias(m *Machine, alias string, src bool, u *User) []string {
	if u != nil && m.HasTags() {
		return []string{}
	}

	if u != nil && !m.HasUser(u.Name) {
		return []string{}
	}

	if alias == "*" && u != nil {
		return m.IPs()
	}

	if alias == "*" {
		return []string{"*"}
	}

	if alias == AutoGroupMembers || alias == AutoGroupSelf {
		if !m.HasTags() {
			return m.IPs()
		} else {
			return []string{}
		}
	}

	if strings.Contains(alias, "@") && !m.HasTags() && m.HasUser(alias) {
		return m.IPs()
	}

	if strings.HasPrefix(alias, "group:") && !m.HasTags() {
		users, ok := a.Groups[alias]

		if !ok {
			return []string{}
		}

		for _, u := range users {
			if m.HasUser(u) {
				return m.IPs()
			}
		}

		return []string{}
	}

	if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
		return m.IPs()
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
	sort.Strings(items)
	return items
}
