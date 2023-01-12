package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func WriteConfig(hosts []models.Host) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, "/.ssh-sync-data")
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {

		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(filepath.Join(p, "config"), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, host := range hosts {
		if _, err := file.WriteString(fmt.Sprintf("Host %s\n", host.Host)); err != nil {
			return err
		}
		if host.IdentityFile != "" {
			if _, err := file.WriteString(fmt.Sprintf("\t%s %s\n", "IdentityFile", filepath.Join(user.HomeDir, host.IdentityFile))); err != nil {
				return err
			}
		}
		for key, value := range host.Values {
			if _, err := file.WriteString(fmt.Sprintf("\t%s %s\n", key, value)); err != nil {
				return err
			}
		}
	}
	return nil
}

func WriteKey(key []byte, filename string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, "/.ssh-sync-data")
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(filepath.Join(p, filename), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(key); err != nil {
		return err
	}
	return nil
}
