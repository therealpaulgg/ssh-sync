package actions

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

// isNonInteractive reports whether the current command should avoid prompting
// the user. It can be forced with --non-interactive/--quiet or when stdin is
// not connected to a terminal.
func isNonInteractive(c *cli.Context) bool {
	if c != nil && (c.Bool("non-interactive") || c.Bool("quiet")) {
		return true
	}
	return !isatty.IsTerminal(os.Stdin.Fd())
}
