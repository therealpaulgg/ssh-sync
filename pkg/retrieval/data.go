package retrieval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func (c RetrievalClient) GetUserData(profile *models.Profile) (dto.DataDto, error) {
	var data dto.DataDto
	token, err := c.GetToken()
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return data, errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return data, err
	}
	masterKey, err := c.RetrieveMasterKey()
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

func (c RetrievalClient) DeleteConfig(profile *models.Profile, configID uuid.UUID) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	dataUrl := profile.ServerUrl
	dataUrl.Path = fmt.Sprintf("/api/v1/data/config/%s", configID)
	req, err := http.NewRequest("DELETE", dataUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to delete config entry. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func (c RetrievalClient) UpsertConfig(profile *models.Profile, config dto.SshConfigDto) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	body, err := json.Marshal(config)
	if err != nil {
		return err
	}
	dataUrl := profile.ServerUrl
	dataUrl.Path = "/api/v1/data/config"
	req, err := http.NewRequest("POST", dataUrl.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to upsert config entry. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func (c RetrievalClient) DeleteKey(profile *models.Profile, key dto.KeyDto) error {
	token, err := c.GetToken()
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to delete data. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func (c RetrievalClient) UploadData(profile *models.Profile, path string, multipartWriter *multipart.Writer, multipartBody bytes.Buffer) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}
	uploadURL := profile.ServerUrl
	uploadURL.Path = "/api/v1/data"
	req, err := http.NewRequest("POST", uploadURL.String(), &multipartBody)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to upload data. status code: " + strconv.Itoa(res.StatusCode))
	}

	var uploadedKeys []dto.KeyDto
	if err := json.NewDecoder(res.Body).Decode(&uploadedKeys); err == nil {
		for _, key := range uploadedKeys {
			if key.UpdatedAt != nil {
				localPath := filepath.Join(path, key.Filename)
				_ = os.Chtimes(localPath, *key.UpdatedAt, *key.UpdatedAt)
			}
		}
	}
	return nil
}
