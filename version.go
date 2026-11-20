package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// version is set at release time with -ldflags "-X main.version=1.2.3".
// Builds from `go install module@version` fall back to the module version
// recorded by the Go toolchain; local builds report "dev".
var version string

func buildVersion() string {
	if version != "" {
		return strings.TrimPrefix(version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

func versionLine() string {
	return fmt.Sprintf("session-ledger %s (%s/%s, %s)", buildVersion(), runtime.GOOS, runtime.GOARCH, runtime.Version())
}
