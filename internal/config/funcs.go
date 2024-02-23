package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

func GetBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if len(v) > 0 {
		return strings.ToLower(v) == "true"
	}
	return defaultValue
}

func GetString(key, defaultValue string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return defaultValue
}

func publicAddrToUrl(addr string) (*url.URL, error) {
	scheme := "https"

	if strings.HasPrefix(addr, "http://") {
		scheme = "http"
		addr = strings.TrimPrefix(addr, "http://")
	}

	if strings.HasPrefix(addr, "https://") {
		scheme = "https"
		addr = strings.TrimPrefix(addr, "https://")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid public addr")
	}

	if (port == "443" && scheme == "https") || (port == "80" && scheme == "http") || port == "" {
		return &url.URL{Scheme: scheme, Host: host}, nil
	}

	return &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%s", host, port)}, nil
}
