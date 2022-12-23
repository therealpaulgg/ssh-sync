package main

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/internal/actions"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "ssh-sync",
		Description: "Syncs your ssh keys to a remote server",
		Commands: []*cli.Command{
			{
				Name:        "setup",
				Description: "Set up your system to use ssh-sync.",
				Action:      actions.Setup,
			},
			{
				Name:   "token",
				Action: actions.Token,
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
				Name:   "parse-config",
				Action: actions.ParseConfig,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
