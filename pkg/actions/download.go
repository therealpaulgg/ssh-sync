package actions

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Download(c *cli.Context) error {
	setup, err := utils.CheckIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	data, err := retrieval.GetUserData(profile)
	if err != nil {
		return err
	}
	isSafeMode := c.Bool("safe-mode")
	var directory string
	if isSafeMode {
		fmt.Println("Executing in safe mode (keys writing to .ssh-sync-data)")
		directory = ".ssh-sync-data"
	} else {
		directory = ".ssh"
	}
	if err := utils.WriteConfig(lo.Map(data.SshConfig, func(config dto.SshConfigDto, i int) models.Host {
		return models.Host{
			Host:          config.Host,
			Values:        config.Values,
			IdentityFiles: config.IdentityFiles,
		}
	}), directory); err != nil {
		return err
	}
	for _, key := range data.Keys {
		if err := utils.WriteKey(key.Data, key.Filename, directory); err != nil {
			return err
		}
	}

	if len(data.KnownHosts) > 0 {
		if err := utils.WriteKnownHosts(data.KnownHosts, directory); err != nil {
			return err
		}
	}

	err = checkForDeletedKeys(data.Keys, directory)

	if err != nil {
		return err
	}
	fmt.Println("Successfully downloaded keys.")
	return nil
}

func checkForDeletedKeys(keys []dto.KeyDto, directory string) error {
	sshDir, err := utils.GetAndCreateSshDirectory(directory)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(sshDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "config" || d.Name() == "known_hosts" {
			return nil
		}
		_, exists := lo.Find(keys, func(key dto.KeyDto) bool {
			return key.Filename == d.Name()
		})
		if exists {
			return nil
		}
		fmt.Printf("Key %s detected on your filesystem that is not in the database. Delete? (y/n): ", d.Name())
		var answer string
		scanner := bufio.NewScanner(os.Stdin)
		if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
			return err
		}
		if answer == "y" {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
