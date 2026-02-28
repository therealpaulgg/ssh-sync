package actions

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

type syncDeps struct {
	checkSetup func() (bool, error)
	getProfile func() (*models.Profile, error)
	getData    func(*models.Profile) (dto.DataDto, error)
	upload     func(*cli.Context) error
	download   func(*cli.Context) error
	decide     func(*cli.Context, dto.DataDto) bool
}

func defaultSyncDeps() syncDeps {
	client := retrieval.NewRetrievalClient()
	return syncDeps{
		checkSetup: utils.CheckIfSetup,
		getProfile: utils.GetProfile,
		getData:    client.GetUserData,
		upload:     Upload,
		download:   Download,
		decide:     shouldDownloadFirst,
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

	profile, err := deps.getProfile()
	if err != nil {
		return err
	}
	data, err := deps.getData(profile)
	if err != nil {
		return err
	}

	if deps.decide(c, data) {
		if err := deps.download(c); err != nil {
			return err
		}
		return deps.upload(c)
	}

	if err := deps.upload(c); err != nil {
		return err
	}
	return deps.download(c)
}

func shouldDownloadFirst(c *cli.Context, data dto.DataDto) bool {
	localPath := c.String("path")
	if localPath == "" {
		currentUser, err := user.Current()
		if err != nil {
			return false
		}
		localPath = filepath.Join(currentUser.HomeDir, ".ssh")
	}
	stat, err := os.Stat(localPath)
	if err != nil {
		return true
	}
	if !stat.IsDir() {
		return true
	}

	serverKeys := map[string]dto.KeyDto{}
	for _, key := range data.Keys {
		if isReservedFilename(key.Filename) {
			continue
		}
		serverKeys[key.Filename] = key
	}

	for _, key := range serverKeys {
		localInfo, err := os.Stat(filepath.Join(localPath, key.Filename))
		if errors.Is(err, os.ErrNotExist) {
			return true
		}
		if err != nil {
			return false
		}
		if key.UpdatedAt != nil && key.UpdatedAt.After(localInfo.ModTime()) {
			return true
		}
	}

	return false
}
