package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func WriteConfig(hosts []models.Host) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := path.Join(user.HomeDir, "/.ssh-sync-data")
	fmt.Println(p)
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(p, 0700)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path.Join(p, "config"), os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, host := range hosts {
		_, err = file.WriteString(fmt.Sprintf("Host %s\n", host.Host))
		if err != nil {
			return err
		}
		for key, value := range host.Values {
			_, err = file.WriteString(fmt.Sprintf("\t%s %s\n", key, value))
			if err != nil {
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
	p := path.Join(user.HomeDir, "/.ssh-sync-data")
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(p, 0700)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path.Join(p, filename), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(key)
	if err != nil {
		return err
	}
	return nil
}
