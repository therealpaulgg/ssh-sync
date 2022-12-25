package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

type DataDto struct {
	ID        uuid.UUID      `json:"id"`
	Username  string         `json:"username"`
	Keys      []KeyDto       `json:"keys"`
	MasterKey []byte         `json:"master_key"`
	SshConfig []SshConfigDto `json:"ssh_config"`
}

type KeyDto struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Filename string    `json:"filename"`
	Data     []byte    `json:"data"`
}

type SshConfigDto struct {
	Host   string            `json:"host"`
	Values map[string]string `json:"values"`
}

func Download(c *cli.Context) error {
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", "http://localhost:3000/api/v1/data", nil)
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
	var data DataDto
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return err
	}
	masterKey, err := utils.Decrypt(data.MasterKey)
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
	err = utils.WriteConfig(lo.Map(data.SshConfig, func(config SshConfigDto, i int) models.Host {
		return models.Host{
			Host:   config.Host,
			Values: config.Values,
		}
	}))
	if err != nil {
		return err
	}
	for _, key := range data.Keys {
		err = utils.WriteKey(key.Data, key.Filename)
		if err != nil {
			return err
		}
	}
	return nil
}
