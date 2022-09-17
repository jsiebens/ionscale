package domain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestACLPolicy_IsTagOwner(t *testing.T) {
	policy := ACLPolicy{
		Groups: map[string][]string{
			"group:engineers": {"jane@example.com"},
		},
		TagOwners: map[string][]string{
			"tag:web": {"john@example.com", "group:engineers"},
		}}

	testCases := []struct {
		name      string
		tag       string
		userName  string
		userType  UserType
		expectErr bool
	}{
		{
			name:      "system admin is always a valid owner",
			tag:       "tag:web",
			userName:  "system admin",
			userType:  UserTypeService,
			expectErr: false,
		},
		{
			name:      "system admin is always a valid owner",
			tag:       "tag:unknown",
			userName:  "system admin",
			userType:  UserTypeService,
			expectErr: false,
		},
		{
			name:      "direct tag owner",
			tag:       "tag:web",
			userName:  "john@example.com",
			userType:  UserTypePerson,
			expectErr: false,
		},
		{
			name:      "owner by group",
			tag:       "tag:web",
			userName:  "jane@example.com",
			userType:  UserTypePerson,
			expectErr: false,
		},
		{
			name:      "unknown owner",
			tag:       "tag:web",
			userName:  "nick@example.com",
			userType:  UserTypePerson,
			expectErr: true,
		},
		{
			name:      "unknown tag",
			tag:       "tag:unknown",
			userName:  "jane@example.com",
			userType:  UserTypePerson,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := policy.CheckTagOwners([]string{tc.tag}, &User{Name: tc.userName, UserType: tc.userType})
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
