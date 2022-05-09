package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type Tags []string

func (i *Tags) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case string:
		t := strings.Trim(value, "|")
		if len(t) == 0 {
			*i = []string{}
		} else {
			*i = strings.Split(t, "|")
		}
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
	return nil
}

func (i Tags) Value() (driver.Value, error) {
	v := "|" + strings.Join(i, "|") + "|"
	return v, nil
}

func SanitizeTags(input []string) Tags {
	keys := make(map[string]bool)
	var tags []string
	for _, v := range input {
		var entry string
		if strings.HasPrefix(v, "tag:") {
			entry = v[4:]
		} else {
			entry = v
		}

		if _, value := keys[entry]; !value {
			keys[entry] = true
			tags = append(tags, entry)
		}
	}
	return tags
}
