package actions

import (
	"os"
	"os/user"
	"path"

	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
	p := c.String("path")
	if p == "" {
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
}
