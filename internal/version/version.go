package version

import "fmt"

var (
	// Version is the semantic version of the application
	Version = "0.0.0"

	// CommitID is the git commit hash
	CommitID = "unknown"

	// BuildDate is the date of the build
	BuildDate = "unknown"
)

type BuildInfo struct {
	Version   string
	BuildDate string
	CommitID  string
}

func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		BuildDate: BuildDate,
		CommitID:  CommitID,
	}
}

func PrintVersion(buildInfo BuildInfo) {
	fmt.Printf("Version: %s\nBuild Date: %s\nCommit ID: %s\n",
		buildInfo.Version, buildInfo.BuildDate, buildInfo.CommitID)
}
