package tsn

import "strings"

type UpFlag = []string

func WithAdvertiseTags(tags string) UpFlag {
	return []string{"--advertise-tags", tags}
}

func WithAdvertiseRoutes(routes []string) UpFlag {
	return []string{"--advertise-routes", strings.Join(routes, ",")}
}
