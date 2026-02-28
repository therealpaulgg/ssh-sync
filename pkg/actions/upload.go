package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
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

	if err := applyPendingRotation(profile); err != nil {
		return fmt.Errorf("applying pending key rotation: %w", err)
	}

	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	url := profile.ServerUrl
	url.Path = "/api/v1/data"
	req, err := http.NewRequest("GET", url.String(), nil)
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
		return errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}

	// Decode the server response to get existing keys with their timestamps
	var serverData dto.DataDto
	if err := json.NewDecoder(res.Body).Decode(&serverData); err != nil {
		return err
	}

	// Create a map of server keys by filename for quick lookup
	serverKeysByFilename := make(map[string]dto.KeyDto)
	for _, key := range serverData.Keys {
		serverKeysByFilename[key.Filename] = key
	}
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return err
	}
	p := c.String("path")
	if p == "" {
		user, err := user.Current()
		if err != nil {
			return err
		}
		p = filepath.Join(user.HomeDir, ".ssh")
	}
	dirEntries, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	keys := make(map[string][]byte)
	hosts := []models.Host{}
	var knownHosts []dto.KnownHostDto

	for _, file := range dirEntries {
		if file.IsDir() || isSkippedBinaryUpload(file.Name()) {
			continue
		} else if file.Name() == "config" {
			hosts, err = utils.ParseConfig()
			if err != nil {
				return err
			}
			if len(hosts) == 0 {
				return errors.New("your ssh config is empty. Please add some hosts to your ssh config so data can be uploaded.")
			}
			continue
		}

		// Get file info to retrieve modification time
		fileInfo, err := file.Info()
		if err != nil {
			return err
		}
		localModTime := fileInfo.ModTime()

		// Check if this key exists on the server with a newer timestamp
		if serverKey, exists := serverKeysByFilename[file.Name()]; exists {
			// Only compare if server has timestamp information
			if serverKey.UpdatedAt != nil && serverKey.UpdatedAt.After(localModTime) {
				// Server key is newer, prompt user
				shouldOverwrite, err := utils.PromptOverwriteNewerKey(file.Name(), localModTime, *serverKey.UpdatedAt)
				if err != nil {
					return err
				}
				if !shouldOverwrite {
					fmt.Printf("Skipping %s\n", file.Name())
					continue
				}
			}
		}

		fileBytes, err := os.ReadFile(filepath.Join(p, file.Name()))
		if err != nil {
			return err
		}
		keys[file.Name()] = fileBytes
	}

	knownHostsPath := filepath.Join(p, "known_hosts")
	if knownHostEntries, err := utils.ParseKnownHosts(knownHostsPath); err == nil && len(knownHostEntries) > 0 {
		knownHosts = knownHostEntriesToDtos(knownHostEntries)
	}

	uploadedKeys, err := sendUpload(keys, hosts, knownHosts, masterKey, token, profile)
	if err != nil {
		return err
	}
	for _, key := range uploadedKeys {
		if key.UpdatedAt != nil {
			localPath := filepath.Join(p, key.Filename)
			_ = os.Chtimes(localPath, *key.UpdatedAt, *key.UpdatedAt)
		}
	}
	fmt.Println("Successfully uploaded keys.")
	return nil
}

// sendUpload encrypts the given keys with masterKey, builds a multipart request,
// POSTs it to /api/v1/data, and returns the uploaded key DTOs (with server timestamps).
func sendUpload(
	keys map[string][]byte,
	hosts []models.Host,
	knownHosts []dto.KnownHostDto,
	masterKey []byte,
	token string,
	profile *models.Profile,
) ([]dto.KeyDto, error) {
	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)

	for filename, data := range keys {
		encBytes, err := utils.EncryptWithMasterKey(data, masterKey)
		if err != nil {
			return nil, err
		}
		w, _ := multipartWriter.CreateFormFile("keys[]", filename)
		if _, err := w.Write(encBytes); err != nil {
			return nil, err
		}
	}
	if len(hosts) > 0 {
		jsonBytes, err := json.Marshal(hosts)
		if err != nil {
			return nil, err
		}
		w, err := multipartWriter.CreateFormField("ssh_config")
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(jsonBytes); err != nil {
			return nil, err
		}
	}
	if len(knownHosts) > 0 {
		jsonBytes, err := json.Marshal(knownHosts)
		if err != nil {
			return nil, err
		}
		w, err := multipartWriter.CreateFormField("known_hosts")
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(jsonBytes); err != nil {
			return nil, err
		}
	}
	multipartWriter.Close()

	postURL := profile.ServerUrl
	postURL.Path = "/api/v1/data"
	req, err := http.NewRequest("POST", postURL.String(), &multipartBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("failed to upload data. status code: " + strconv.Itoa(res.StatusCode))
	}
	var uploadedKeys []dto.KeyDto
	if err := json.NewDecoder(res.Body).Decode(&uploadedKeys); err != nil {
		return nil, err
	}
	return uploadedKeys, nil
}

// knownHostEntriesToDtos converts a slice of local KnownHostEntry models to the
// wire DTO form used by the server API.
func knownHostEntriesToDtos(entries []models.KnownHostEntry) []dto.KnownHostDto {
	dtos := make([]dto.KnownHostDto, len(entries))
	for i, e := range entries {
		dtos[i] = dto.KnownHostDto{
			HostPattern: e.HostPattern,
			KeyType:     e.KeyType,
			KeyData:     e.KeyData,
			Marker:      e.Marker,
		}
	}
	return dtos
}

// isSkippedBinaryUpload reports whether a filename must not be sent as an
// encrypted binary key. known_hosts is synced as structured entries;
// authorized_keys must never leave the local machine.
func isSkippedBinaryUpload(name string) bool {
	switch name {
	case "known_hosts", "authorized_keys":
		return true
	}
	return false
}
