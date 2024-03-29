package main

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/actions"
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
				Name:   "download",
				Action: actions.Download,
			},
			{
				Name:   "challenge-response",
				Action: actions.ChallengeResponse,
			},
			{
				Name:   "list-machines",
				Action: actions.ListMachines,
			},
			{
				Name:   "remove-machine",
				Action: actions.RemoveMachine,
			},
			{
				Name:   "reset",
				Action: actions.Reset,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
