package version

var (
	Version    string
	Revision   string
	DevVersion = "dev"
)

func BuildVersion() string {
	if len(Version) == 0 {
		return DevVersion
	}
	return Version
}

func GetReleaseInfo() (string, string) {
	return BuildVersion(), Revision
}
