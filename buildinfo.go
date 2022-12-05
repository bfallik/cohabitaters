package cohabitaters

import (
	"fmt"
	"runtime/debug"
)

func BuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "missing build info"
	}

	vcs := "(unknown vcs)"
	buildRev := "(unknown revision)"
	buildTime := "(unknown time)"

	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs":
			vcs = kv.Value
		case "vcs.revision":
			buildRev = kv.Value
		case "vcs.time":
			buildTime = kv.Value
		}
	}

	return fmt.Sprintf("%s rev: %s built at %s", vcs, buildRev, buildTime)
}
