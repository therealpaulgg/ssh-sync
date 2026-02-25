package retrieval

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func TestGetMachines(t *testing.T) {
	// Arrange
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]dto.MachineDto{
			{
				Name: "test",
			},
		})
	}))
	defer server.Close()
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	machines, err := GetMachines(profile)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, len(machines))
	assert.Equal(t, "test", machines[0].Name)
}

func TestDeleteMachine(t *testing.T) {
	// Arrange
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	err := DeleteMachine(profile, "test")
	// Assert
	assert.Nil(t, err)
}

func TestDeleteMachineDoesNotExist(t *testing.T) {
	// Arrange
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	err := DeleteMachine(profile, "test")
	// Assert
	assert.NotNil(t, err)
}
