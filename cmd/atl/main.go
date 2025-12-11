package main

import (
	"os"

	"github.com/enthus-appdev/atl-cli/internal/cmd"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

// Build information set by ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	buildInfo := cmd.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	ios := iostreams.System()
	code := cmd.Execute(ios, buildInfo)
	os.Exit(code)
}
