package main

import (
	"os"

	"github.com/rybesh/zulip-cli/cmd/zulip-cli/commands"
)

func main() {
	// Cobra has already reported the error to stderr, along with usage when the
	// failure was a usage error; printing it again only duplicates it.
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
