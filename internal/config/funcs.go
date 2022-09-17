package config

import (
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
