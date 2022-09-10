package config

import (
	"os"
	"strings"
	"time"
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

func GetStrings(key string, defaultValue []string) []string {
	v := os.Getenv(key)
	if v != "" {
		return strings.Split(v, ",")
	}
	return defaultValue
}

func GetDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if len(v) > 0 {
		duration, err := time.ParseDuration(v)
		if err != nil {
			return defaultValue, err
		}
		return duration, nil
	}
	return defaultValue, nil
}
