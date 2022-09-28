package domain

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizeTailnetName(t *testing.T) {
	assert.Equal(t, "john.example.com", SanitizeTailnetName("john@example.com"))
	assert.Equal(t, "john.example.com", SanitizeTailnetName("john@examPle.Com"))
	assert.Equal(t, "john-doe.example.com", SanitizeTailnetName("john.doe@example.com"))
	assert.Equal(t, "johns-network", SanitizeTailnetName("John's Network"))
	assert.Equal(t, "example.com", SanitizeTailnetName("example.com"))
	assert.Equal(t, "johns-example.com", SanitizeTailnetName("John's example.com"))
}
