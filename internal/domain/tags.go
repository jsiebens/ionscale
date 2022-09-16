package domain

import (
	"database/sql/driver"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"strings"
	"tailscale.com/tailcfg"
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
	if len(i) == 0 {
		return "", nil
	}
	v := "|" + strings.Join(i, "|") + "|"
	return v, nil
}

func CheckTag(tag string) error {
	return tailcfg.CheckTag(tag)
}

func CheckTags(tags []string) error {
	var result *multierror.Error
	for _, t := range tags {
		if err := CheckTag(t); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

func SanitizeTags(input []string) Tags {
	s := StringSet{}
	return s.Add(input...).Items()
}
