package main

import (
	"fmt"
	"os"

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
				Action:      actions.Setup,
			},
			{
				Name: "upload",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
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
				},
				Action: actions.Download,
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
