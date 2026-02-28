package actions

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Download(c *cli.Context) error {
	opts := downloadOptions{
		SafeMode:       c.Bool("safe-mode"),
		NonInteractive: isNonInteractive(c),
	}
	return runDownload(opts)
}

type downloadOptions struct {
	SafeMode       bool
	NonInteractive bool
}

func runDownload(opts downloadOptions) error {
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
	client := retrieval.NewRetrievalClient()
	data, err := client.GetUserData(profile)
	if err != nil {
		return err
	}
	var directory string
	if opts.SafeMode {
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
		if isReservedFilename(key.Filename) {
			continue
		}
		if err := utils.WriteKey(key.Data, key.Filename, directory, opts.NonInteractive); err != nil {
			return err
		}
	}
	if len(data.KnownHosts) > 0 {
		entries := lo.Map(data.KnownHosts, func(kh dto.KnownHostDto, _ int) models.KnownHostEntry {
			return models.KnownHostEntry{
				HostPattern: kh.HostPattern,
				KeyType:     kh.KeyType,
				KeyData:     kh.KeyData,
				Marker:      kh.Marker,
			}
		})
		if err := utils.WriteKnownHosts(entries, directory); err != nil {
			return err
		}
	}

	err = checkForDeletedKeys(data.Keys, directory, opts.NonInteractive)

	if err != nil {
		return err
	}
	fmt.Println("Successfully downloaded keys.")
	return nil
}

func checkForDeletedKeys(keys []dto.KeyDto, directory string, nonInteractive bool) error {
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
		if isReservedFilename(d.Name()) {
			return nil
		}
		_, exists := lo.Find(keys, func(key dto.KeyDto) bool {
			return key.Filename == d.Name()
		})
		if exists {
			return nil
		}
		if nonInteractive {
			fmt.Fprintf(os.Stderr, "Non-interactive mode: %s exists locally but not on server; leaving untouched.\n", d.Name())
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

// isReservedFilename reports whether a filename should never be treated as a
// synced key — either because it is managed separately (config, known_hosts)
// or because it must never leave the local machine (authorized_keys).
func isReservedFilename(name string) bool {
	switch name {
	case "known_hosts", "authorized_keys", "config":
		return true
	}
	return false
}
