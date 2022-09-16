package domain

type Principal struct {
	SystemRole SystemRole
	User       *User
	UserRole   UserRole
}

func (p Principal) IsSystemAdmin() bool {
	return p.SystemRole.IsAdmin()
}

func (p Principal) IsTailnetAdmin(tailnetID uint64) bool {
	return p.User.TailnetID == tailnetID && p.UserRole.IsAdmin()
}

func (p Principal) IsTailnetMember(tailnetID uint64) bool {
	return p.User.TailnetID == tailnetID
}

func (p Principal) UserMatches(userID uint64) bool {
	return p.User.ID == userID
}
