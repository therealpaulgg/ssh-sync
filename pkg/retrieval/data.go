package retrieval

import (
	"encoding/json"
	"errors"
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
	return data, nil
}
