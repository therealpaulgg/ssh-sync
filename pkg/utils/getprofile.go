package utils

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func GetProfile() (*models.Profile, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "profile.json")
	jsonBytes, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var profile models.Profile
	if err := json.Unmarshal(jsonBytes, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
