package domain

import (
	"context"
	"github.com/jsiebens/ionscale/internal/util"
)

type SystemRole string

const (
	SystemRoleNone  SystemRole = ""
	SystemRoleAdmin SystemRole = "admin"
)

func (s SystemRole) IsAdmin() bool {
	return s == SystemRoleAdmin
}

type TailnetRole string

const (
	TailnetRoleService TailnetRole = "service"
	TailnetRoleMember  TailnetRole = "member"
)

type User struct {
	ID   uint64 `gorm:"primary_key;autoIncrement:false"`
	Name string

	TailnetRole TailnetRole
	TailnetID   uint64
	Tailnet     Tailnet

	AccountID *uint64
	Account   *Account
}

type Users []User

func (r *repository) GetOrCreateServiceUser(ctx context.Context, tailnet *Tailnet) (*User, bool, error) {
	user := &User{}
	id := util.NextID()

	query := User{Name: tailnet.Name, TailnetID: tailnet.ID, TailnetRole: TailnetRoleService}
	attrs := User{ID: id, Name: tailnet.Name, TailnetID: tailnet.ID, TailnetRole: TailnetRoleService}

	tx := r.withContext(ctx).Where(query).Attrs(attrs).FirstOrCreate(user)

	if tx.Error != nil {
		return nil, false, tx.Error
	}

	return user, user.ID == id, nil
}

func (r *repository) ListUsers(ctx context.Context, tailnetID uint64) (Users, error) {
	var users = []User{}

	tx := r.withContext(ctx).Where("tailnet_id = ?", tailnetID).Find(&users)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return users, nil
}

func (r *repository) DeleteUsersByTailnet(ctx context.Context, tailnetID uint64) error {
	tx := r.withContext(ctx).Where("tailnet_id = ?", tailnetID).Delete(&User{})
	return tx.Error
}

func (r *repository) GetOrCreateUserWithAccount(ctx context.Context, tailnet *Tailnet, account *Account) (*User, bool, error) {
	user := &User{}
	id := util.NextID()

	query := User{AccountID: &account.ID, TailnetID: tailnet.ID}
	attrs := User{ID: id, Name: account.LoginName, TailnetID: tailnet.ID, AccountID: &account.ID, TailnetRole: TailnetRoleMember}

	tx := r.withContext(ctx).Where(query).Attrs(attrs).FirstOrCreate(user)

	if tx.Error != nil {
		return nil, false, tx.Error
	}

	return user, user.ID == id, nil
}
