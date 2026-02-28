package actions

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

type syncDeps struct {
	checkSetup func() (bool, error)
	upload     func(*cli.Context) error
	download   func(*cli.Context) error
}

func defaultSyncDeps() syncDeps {
	return syncDeps{
		checkSetup: utils.CheckIfSetup,
		upload:     Upload,
		download:   Download,
	}
}

func Sync(c *cli.Context) error {
	return syncWithDeps(c, defaultSyncDeps())
}

func syncWithDeps(c *cli.Context, deps syncDeps) error {
	setup, err := deps.checkSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}

	if err := deps.upload(c); err != nil {
		return err
	}
	return deps.download(c)
}
