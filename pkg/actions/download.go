package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Download(c *cli.Context) error {
	setup, err := checkIfSetup()
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
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	dataUrl := profile.ServerUrl
	dataUrl.Path = "/api/v1/data"
	req, err := http.NewRequest("GET", dataUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}
	var data dto.DataDto
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return err
	}
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return err
	}
	for i, key := range data.Keys {
		decryptedKey, err := utils.DecryptWithMasterKey(key.Data, masterKey)
		if err != nil {
			return err
		}
		data.Keys[i].Data = decryptedKey
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
	fmt.Println("Successfully downloaded keys.")
	return nil
}
