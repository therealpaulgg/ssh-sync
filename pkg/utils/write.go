package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func WriteConfig(hosts []models.Host, sshDirectory string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, sshDirectory)
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {

		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(filepath.Join(p, "config"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, host := range hosts {
		if _, err := file.WriteString(fmt.Sprintf("Host %s\n", host.Host)); err != nil {
			return err
		}
		if host.IdentityFiles != nil {
			for _, identityFile := range host.IdentityFiles {
				if _, err := file.WriteString(fmt.Sprintf("\t%s %s\n", "IdentityFile", filepath.Join(user.HomeDir, identityFile))); err != nil {
					return err
				}
			}
		}
		for key, value := range host.Values {
			for _, item := range value {
				if _, err := file.WriteString(fmt.Sprintf("\t%s %s\n", key, item)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func WriteKey(key []byte, filename string, sshDirectory string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, sshDirectory)
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	_, err = os.OpenFile(filepath.Join(p, filename), os.O_RDONLY, 0600)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		existingData, err := os.ReadFile(filepath.Join(p, filename))
		if err != nil {
			return err
		}
		if string(existingData) != string(key) {
			var answer string
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Printf("diff detected for %s.\n", filename)
			fmt.Println("1. Overwrite")
			fmt.Println("2. Skip")
			fmt.Println("3. Save new file (as .duplicate extension for manual resolution)")
			fmt.Print("Please choose an option (will skip by default): ")

			if err := ReadLineFromStdin(scanner, &answer); err != nil {
				return err
			}
			fmt.Println()
			if answer == "3" {
				filename = filename + ".duplicate"
			} else if answer == "2" || answer != "1" {
				return nil
			}
		}
	}
	file, err := os.OpenFile(filepath.Join(p, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(key); err != nil {
		return err
	}

	return nil
}
