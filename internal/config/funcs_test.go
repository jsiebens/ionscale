package config

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestPublicAddrToUrl(t *testing.T) {
	mustParseUrl := func(s string) *url.URL {
		parse, err := url.Parse(s)
		require.NoError(t, err)
		return parse
	}

	parameters := []struct {
		input    string
		expected *url.URL
		err      error
	}{
		{"localtest.me", nil, fmt.Errorf("invalid public addr")},
		{"localtest.me:443", mustParseUrl("https://localtest.me"), nil},
		{"localtest.me:80", mustParseUrl("https://localtest.me:80"), nil},
		{"localtest.me:8080", mustParseUrl("https://localtest.me:8080"), nil},
		{"http://localtest.me:8080", mustParseUrl("http://localtest.me:8080"), nil},
	}

	for _, p := range parameters {
		t.Run(fmt.Sprintf("Testing [%v]", p.input), func(t *testing.T) {
			url, err := publicAddrToUrl(p.input)
			require.Equal(t, p.expected, url)
			require.Equal(t, p.err, err)
		})
	}
}
