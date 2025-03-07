package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
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

func validatePublicAddr(addr string) (*url.URL, string, int, error) {
	scheme := "https"

	if strings.HasPrefix(addr, "http://") {
		scheme = "http"
		addr = strings.TrimPrefix(addr, "http://")
	}

	if strings.HasPrefix(addr, "https://") {
		scheme = "https"
		addr = strings.TrimPrefix(addr, "https://")
	}

	host, portS, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, "", -1, fmt.Errorf("invalid")
	}

	port, err := strconv.Atoi(portS)
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid")
	}

	if (port == 443 && scheme == "https") || (port == 80 && scheme == "http") {
		return &url.URL{Scheme: scheme, Host: host}, host, port, nil
	}

	return &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", host, port)}, host, port, nil
}
