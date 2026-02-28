package main

import (
	"fmt"
	"os"
	"time"

	"github.com/therealpaulgg/ssh-sync/pkg/actions"
	"github.com/therealpaulgg/ssh-sync/pkg/actions/interactive"
	"github.com/urfave/cli/v2"
)

var version string

func main() {
	app := &cli.App{
		Name:        "ssh-sync",
		Usage:       "sync your ssh keys to a remote server",
		Version:     version,
		Description: "Syncs your ssh keys to a remote server",
		Commands: []*cli.Command{
			{
				Name:        "setup",
				Description: "Set up your system to use ssh-sync.",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "classic",
						Aliases: []string{"c"},
						Usage:   "Use classical elliptic curve cryptography (ECDSA/ECDH-ES) instead of post-quantum",
					},
				},
				Action: actions.Setup,
			},
			{
				Name: "upload",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
					},
					&cli.BoolFlag{
						Name:    "non-interactive",
						Aliases: []string{"q", "quiet"},
						Usage:   "Run without prompts; skip conflicts instead of overwriting",
					},
				},
				Action: actions.Upload,
			},
			{
				Name: "download",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "safe-mode",
						Aliases: []string{"s"},
						Usage:   "Safe mode will sync to an alternate directory (.ssh-sync-data)",
					},
					&cli.BoolFlag{
						Name:    "non-interactive",
						Aliases: []string{"q", "quiet"},
						Usage:   "Run without prompts; skip conflicts instead of overwriting",
					},
				},
				Action: actions.Download,
			},
			{
				Name:        "sync",
				Description: "Upload local SSH data then download the latest from the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
					},
					&cli.BoolFlag{
						Name:    "safe-mode",
						Aliases: []string{"s"},
						Usage:   "Safe mode will sync to an alternate directory (.ssh-sync-data)",
					},
					&cli.BoolFlag{
						Name:    "non-interactive",
						Aliases: []string{"q", "quiet"},
						Usage:   "Run without prompts; skip conflicts instead of overwriting",
					},
				},
				Action: actions.Sync,
			},
			{
				Name:        "daemon",
				Description: "Continuously run sync on a schedule",
				Flags: []cli.Flag{
					&cli.DurationFlag{
						Name:    "interval",
						Aliases: []string{"i"},
						Usage:   "Interval between sync runs",
						Value:   time.Hour,
					},
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
					},
					&cli.BoolFlag{
						Name:    "safe-mode",
						Aliases: []string{"s"},
						Usage:   "Safe mode will sync to an alternate directory (.ssh-sync-data)",
					},
					&cli.BoolFlag{
						Name:    "non-interactive",
						Aliases: []string{"q", "quiet"},
						Usage:   "Run without prompts; skip conflicts instead of overwriting",
					},
				},
				Action: actions.Daemon,
			},
			{
				Name:      "challenge-response",
				ArgsUsage: "[challenge-phrase]",
				Action:    actions.ChallengeResponse,
			},
			{
				Name:   "list-machines",
				Action: actions.ListMachines,
			},
			{
				Name:      "remove-machine",
				ArgsUsage: "[machine-name]",
				Action:    actions.RemoveMachine,
			},
			{
				Name:   "reset",
				Action: actions.Reset,
			},
			{
				Name:        "migrate",
				Description: "Migrate keys from classical ECDSA to post-quantum (ML-DSA-65 + ML-KEM-768)",
				Action:      actions.Migrate,
			},
			{
				Name:        "interactive",
				Description: "Uses a TUI mode for interacting with keys and config",
				Usage:       "Interactively manage your ssh keys with a TUI",
				Action:      interactive.Interactive,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
