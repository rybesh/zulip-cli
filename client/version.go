package client

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// ldflagsVersion can be set at link time when a precise version is needed for a
// build that has no module version to read:
//
//	go build -ldflags "-X github.com/rybesh/zulip-cli/client.ldflagsVersion=v0.2.0"
//
// When empty, the version comes from the build information Go embeds
// automatically, which is the normal case.
var ldflagsVersion string

// ClientVersion is the version of this build. It is sent to the server in the
// User-Agent header on every request, so server-side logs can tell one build of
// this tool from another.
var ClientVersion = clientVersion(CurrentBuild())

// BuildInfo describes what this binary was built from.
type BuildInfo struct {
	Version   string `json:"version"`
	Revision  string `json:"revision,omitempty"`
	Time      string `json:"time,omitempty"`
	Modified  bool   `json:"modified,omitempty"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// CurrentBuild reports the provenance of this binary.
//
// A binary produced by `go install module@version` records its module version
// in the embedded build information, and that version is reported as-is. A
// local `go build` has no module version, so the version reads "devel" and the
// VCS revision Go embeds is reported alongside it.
func CurrentBuild() BuildInfo {
	b := BuildInfo{
		Version:   "devel",
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		// Go reports "(devel)" for a build with no module version.
		if v := info.Main.Version; v != "" && v != "(devel)" {
			b.Version = v
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				b.Revision = s.Value
			case "vcs.time":
				b.Time = s.Value
			case "vcs.modified":
				b.Modified = s.Value == "true"
			}
		}
	}

	// An explicit link-time version wins over anything inferred.
	if ldflagsVersion != "" {
		b.Version = ldflagsVersion
	}

	return b
}

// clientVersion renders a BuildInfo as the single token reported in the
// User-Agent header. Released builds report their tag; development builds
// carry the revision, so that two untagged builds remain distinguishable in a
// server's request log.
func clientVersion(b BuildInfo) string {
	v := b.Version
	if v != "devel" || b.Revision == "" {
		return v
	}

	v += "+" + shortRevision(b.Revision)
	if b.Modified {
		v += ".dirty"
	}
	return v
}

// shortRevision abbreviates a commit hash the way git does.
func shortRevision(rev string) string {
	const n = 7
	if len(rev) <= n {
		return rev
	}
	// Guard against a revision that is not a plain hex hash.
	if strings.ContainsAny(rev[:n], " \t") {
		return rev
	}
	return rev[:n]
}
