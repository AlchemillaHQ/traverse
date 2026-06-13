package version

var (
	Version   = "0.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return Version
}

func Full() string {
	return Version + " (" + Commit + ") built " + BuildDate
}
