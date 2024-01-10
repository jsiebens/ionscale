package tsn

type UpFlag = []string

func WithAdvertiseTags(tags string) UpFlag {
	return []string{"--advertise-tags", tags}
}
