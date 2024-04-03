package retrieval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func GetMachines(profile *models.Profile) ([]dto.MachineDto, error) {
	url := profile.ServerUrl
	url.Path = "/api/v1/machines/"
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	token, err := utils.GetToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	machines := []dto.MachineDto{}
	err = json.NewDecoder(resp.Body).Decode(&machines)
	if err != nil {
		return nil, err
	}
	return machines, nil
}

func DeleteMachine(profile *models.Profile, machineName string) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(dto.MachineDto{
		Name: machineName,
	}); err != nil {
		return err
	}
	url := profile.ServerUrl
	url.Path = "/api/v1/machines/"
	req, err := http.NewRequest("DELETE", url.String(), buf)
	if err != nil {
		return err
	}
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
