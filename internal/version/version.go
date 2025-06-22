package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

const (
	// VCS設定キー
	vcsRevisionKey = "vcs.revision"
	vcsModifiedKey = "vcs.modified"
	vcsTimeKey     = "vcs.time"

	// デフォルト値
	unknownValue = "unknown"
	devVersion   = "dev"
	develVersion = "(devel)"
)

var (
	// これらの変数はビルド時にldflags経由で設定される
	Version   = devVersion   // バージョン番号
	GitCommit = unknownValue // Gitコミットハッシュ
	GitTag    = ""           // Gitタグ
	BuildDate = unknownValue // ビルド日時
	GoVersion = runtime.Version()
)

// GetVersion はバージョン情報を返す
func GetVersion() string {
	// ldflags経由で設定されたバージョンを優先
	if GitTag != "" {
		return GitTag
	}
	if Version != devVersion {
		return Version
	}

	// go installの場合はdebug.BuildInfoから取得
	if info, ok := debug.ReadBuildInfo(); ok {
		// モジュールバージョンを確認
		if info.Main.Version != "" && info.Main.Version != develVersion {
			return info.Main.Version
		}

		// VCS情報からバージョンを構築
		var revision, modified string
		for _, setting := range info.Settings {
			switch setting.Key {
			case vcsRevisionKey:
				revision = setting.Value
			case vcsModifiedKey:
				modified = setting.Value
			}
		}

		if revision != "" {
			version := revision
			if len(revision) > 7 {
				version = revision[:7]
			}
			if modified == "true" {
				version += "+dirty"
			}
			return version
		}
	}

	return Version
}

// GetBuildInfo は詳細なビルド情報を返す
func GetBuildInfo() string {
	version := GetVersion()

	// ldflags経由のコミット情報を優先
	if GitCommit != unknownValue && len(GitCommit) > 7 {
		version += fmt.Sprintf(" (commit: %s)", GitCommit[:7])
	} else if info, ok := debug.ReadBuildInfo(); ok {
		// VCS情報からコミット情報を取得
		for _, setting := range info.Settings {
			if setting.Key == vcsRevisionKey && len(setting.Value) > 7 {
				version += fmt.Sprintf(" (commit: %s)", setting.Value[:7])
				break
			}
		}
	}

	buildDate := BuildDate
	if buildDate == unknownValue {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == vcsTimeKey {
					buildDate = setting.Value
					break
				}
			}
		}
	}

	return fmt.Sprintf(`tf-file-organize %s
Built with: %s
Build date: %s`, version, GoVersion, buildDate)
}
