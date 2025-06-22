package version

import (
	"fmt"
	"runtime"
)

var (
	// これらの変数はビルド時にldflags経由で設定される
	Version   = "dev"     // バージョン番号
	GitCommit = "unknown" // Gitコミットハッシュ
	GitTag    = ""        // Gitタグ
	BuildDate = "unknown" // ビルド日時
	GoVersion = runtime.Version()
)

// GetVersion はバージョン情報を返す
func GetVersion() string {
	if GitTag != "" {
		return GitTag
	}
	return Version
}

// GetBuildInfo は詳細なビルド情報を返す
func GetBuildInfo() string {
	version := GetVersion()
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		version += fmt.Sprintf(" (commit: %s)", GitCommit[:7])
	}

	return fmt.Sprintf(`terraform-file-organize %s
Built with: %s
Build date: %s`, version, GoVersion, BuildDate)
}
