package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"inet.af/netaddr"
	"tailscale.com/tailcfg"
	"time"
)

type Machine struct {
	ID             uint64 `gorm:"primary_key;autoIncrement:false"`
	Name           string
	NameIdx        uint64
	MachineKey     string
	NodeKey        string
	DiscoKey       string
	Ephemeral      bool
	RegisteredTags Tags
	Tags           Tags

	HostInfo  HostInfo
	Endpoints Endpoints
	AllowIPs  AllowIPs

	IPv4 IP
	IPv6 IP

	CreatedAt time.Time
	ExpiresAt *time.Time
	LastSeen  *time.Time

	UserID uint64
	User   User

	TailnetID uint64
	Tailnet   Tailnet
}

type Machines []Machine

func (m *Machine) IsExpired() bool {
	return m.ExpiresAt != nil && !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(time.Now())
}

func (m *Machine) HasIP(v netaddr.IP) bool {
	return v.Compare(*m.IPv4.IP) == 0 || v.Compare(*m.IPv6.IP) == 0
}

func (m *Machine) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (m *Machine) HasUser(loginName string) bool {
	return m.User.Name == loginName
}

func (m *Machine) HasTags() bool {
	return len(m.Tags) != 0
}

func (m *Machine) IsAllowedIP(i netaddr.IP) bool {
	if m.HasIP(i) {
		return true
	}
	for _, t := range m.AllowIPs {
		if t.Contains(i) {
			return true
		}
	}
	return false
}

func (m *Machine) IsAllowedIPPrefix(i netaddr.IPPrefix) bool {
	for _, t := range m.AllowIPs {
		if t.Overlaps(i) {
			return true
		}
	}
	return false
}

type IP struct {
	*netaddr.IP
}

func (i *IP) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case string:
		ip, err := netaddr.ParseIP(value)
		if err != nil {
			return err
		}
		*i = IP{&ip}
		return nil
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i IP) Value() (driver.Value, error) {
	if i.IP == nil {
		return nil, nil
	}
	return i.String(), nil
}

type AllowIPs []netaddr.IPPrefix

func (hi *AllowIPs) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, hi)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (hi AllowIPs) Value() (driver.Value, error) {
	bytes, err := json.Marshal(hi)
	return bytes, err
}

// GormDataType gorm common data type
func (AllowIPs) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (AllowIPs) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

type HostInfo tailcfg.Hostinfo

func (hi *HostInfo) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, hi)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (hi HostInfo) Value() (driver.Value, error) {
	bytes, err := json.Marshal(hi)
	return bytes, err
}

// GormDataType gorm common data type
func (HostInfo) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (HostInfo) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

type Endpoints []string

func (hi *Endpoints) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, hi)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (hi Endpoints) Value() (driver.Value, error) {
	bytes, err := json.Marshal(hi)
	return bytes, err
}

// GormDataType gorm common data type
func (Endpoints) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (Endpoints) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

func (r *repository) SaveMachine(ctx context.Context, machine *Machine) error {
	tx := r.withContext(ctx).Save(machine)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) DeleteMachine(ctx context.Context, id uint64) (bool, error) {
	tx := r.withContext(ctx).Delete(&Machine{}, id)
	return tx.RowsAffected == 1, tx.Error
}

func (r *repository) GetMachine(ctx context.Context, machineID uint64) (*Machine, error) {
	var m Machine
	tx := r.withContext(ctx).Preload("Tailnet").Preload("User").First(&m, "id = ?", machineID)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) GetNextMachineNameIndex(ctx context.Context, tailnetID uint64, name string) (uint64, error) {
	var m Machine

	tx := r.withContext(ctx).
		Where("name = ? AND tailnet_id = ?", name, tailnetID).
		Order("name_idx desc").
		First(&m)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	}

	if tx.Error != nil {
		return 0, tx.Error
	}

	return m.NameIdx + 1, nil
}

func (r *repository) GetMachineByKey(ctx context.Context, tailnetID uint64, machineKey string) (*Machine, error) {
	var m Machine
	tx := r.withContext(ctx).Preload("Tailnet").Preload("User").First(&m, "tailnet_id = ? AND machine_key = ?", tailnetID, machineKey)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) GetMachineByKeys(ctx context.Context, machineKey string, nodeKey string) (*Machine, error) {
	var m Machine
	tx := r.withContext(ctx).Preload("Tailnet").Preload("User").First(&m, "machine_key = ? AND node_key = ?", machineKey, nodeKey)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) CountMachinesWithIPv4(ctx context.Context, ip string) (int64, error) {
	var count int64

	tx := r.withContext(ctx).Model(&Machine{}).Where("ipv4 = ?", ip).Count(&count)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

func (r *repository) CountMachineByTailnet(ctx context.Context, tailnetID uint64) (int64, error) {
	var count int64

	tx := r.withContext(ctx).Model(&Machine{}).Where("tailnet_id = ?", tailnetID).Count(&count)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return count, nil
}

func (r *repository) DeleteMachineByTailnet(ctx context.Context, tailnetID uint64) error {
	tx := r.withContext(ctx).Model(&Machine{}).Where("tailnet_id = ?", tailnetID).Delete(&Machine{})
	return tx.Error
}

func (r *repository) ListMachineByTailnet(ctx context.Context, tailnetID uint64) (Machines, error) {
	var machines = []Machine{}

	tx := r.withContext(ctx).
		Preload("Tailnet").
		Preload("User").
		Where("tailnet_id = ?", tailnetID).
		Order("name asc, name_idx asc").
		Find(&machines)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return machines, nil
}

func (r *repository) ListMachinePeers(ctx context.Context, tailnetID uint64, key string) (Machines, error) {
	var machines = []Machine{}

	tx := r.withContext(ctx).
		Preload("Tailnet").
		Preload("User").
		Where("tailnet_id = ? AND machine_key <> ?", tailnetID, key).
		Order("id asc").
		Find(&machines)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return machines, nil
}

func (r *repository) ListInactiveEphemeralMachines(ctx context.Context, t time.Time) (Machines, error) {
	var machines = []Machine{}

	tx := r.withContext(ctx).
		Where("ephemeral = ? AND last_seen < ?", true, t.UTC()).
		Find(&machines)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return machines, nil
}

func (r *repository) SetMachineLastSeen(ctx context.Context, machineID uint64) error {
	now := time.Now().UTC()
	tx := r.withContext(ctx).Model(Machine{}).Where("id = ?", machineID).Updates(map[string]interface{}{"last_seen": &now})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) ExpireMachineByAuthMethod(ctx context.Context, authMethodID uint64) (int64, error) {
	now := time.Now().UTC()

	subQuery := r.withContext(ctx).
		Select("machines.id").
		Table("machines").
		Joins("JOIN users u on u.id = machines.user_id JOIN accounts a on a.id = u.account_id").
		Where("a.auth_method_id = ?", authMethodID)

	tx := r.withContext(ctx).
		Table("machines").
		Where("tags = '' AND (expires_at is null or expires_at > ?) AND id in (?)", &now, subQuery).
		Updates(map[string]interface{}{"expires_at": &now})

	if tx.Error != nil {
		return 0, tx.Error
	}

	return tx.RowsAffected, nil
}
