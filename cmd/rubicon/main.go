// Command rubicon is the decision-card and session-handoff CLI for AI agents.
package main

import (
	"os"

	"github.com/gitstq/Rubicon/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
