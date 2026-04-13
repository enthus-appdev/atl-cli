package main

import (
	"os"

	"github.com/enthus-appdev/atl-cli/internal/cmd"
	"github.com/enthus-appdev/atl-cli/internal/iostreams"
)

var version = "dev"

func main() {
	ios := iostreams.System()
	code := cmd.Execute(ios, version)
	os.Exit(code)
}
