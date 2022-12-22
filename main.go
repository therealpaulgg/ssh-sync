package main

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "ssh-sync",
		Description: "Syncs your ssh keys to a remote server",
		Commands: []*cli.Command{
			{
				Name: "upload",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
					},
				},
				Action: func(c *cli.Context) error {
					// get path flag
					p := c.String("path")
					if p == "" {
						// get current user
						user, err := user.Current()
						if err != nil {
							return err
						}
						p = path.Join(user.HomeDir, ".ssh")
					}
					data, err := os.ReadDir(p)
					if err != nil {
						return err
					}
					for _, file := range data {
						if file.IsDir() {
							continue
						}
						println(file.Name())
					}
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
