package retrieval

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func GetUserData(profile *models.Profile) (dto.DataDto, error) {
	var data dto.DataDto
	token, err := utils.GetToken()
	if err != nil {
		return data, err
	}
	dataUrl := profile.ServerUrl
	dataUrl.Path = "/api/v1/data"
	req, err := http.NewRequest("GET", dataUrl.String(), nil)
	if err != nil {
		return data, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return data, err
	}
	if res.StatusCode != 200 {
		return data, errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return data, err
	}
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return data, err
	}
	for i, key := range data.Keys {
		decryptedKey, err := utils.DecryptWithMasterKey(key.Data, masterKey)
		if err != nil {
			return data, err
		}
		data.Keys[i].Data = decryptedKey
	}
	
	if len(data.KnownHosts) > 0 {
		decryptedKnownHosts, err := utils.DecryptWithMasterKey(data.KnownHosts, masterKey)
		if err != nil {
			return data, err
		}
		data.KnownHosts = decryptedKnownHosts
	}
	
	return data, nil
}

func DeleteKey(profile *models.Profile, key dto.KeyDto) error {
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	dataUrl := profile.ServerUrl
	dataUrl.Path = fmt.Sprintf("/api/v1/data/key/%s", key.ID)
	req, err := http.NewRequest("DELETE", dataUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("failed to delete data. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}
