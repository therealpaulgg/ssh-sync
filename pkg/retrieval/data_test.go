package retrieval_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
)

func TestDownloadData(t *testing.T) {
	// Arrange
	// TODO generate ecdsa key
	// TODO encrypt key with a user's private key
	key := []byte{}
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]dto.DataDto{
			{
				ID:       uuid.New(),
				Username: "test",
				Keys: []dto.KeyDto{
					{
						ID:       uuid.New(),
						UserID:   uuid.New(),
						Filename: "test",
						Data:     key,
					},
				},
				SshConfig: []dto.SshConfigDto{
					{
						Host: "test",
						Values: map[string][]string{
							"foo": {"bar"},
						},
						IdentityFiles: []string{"test"},
					},
				},
			},
		})
	}))
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	data, err := retrieval.GetUserData(profile)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, len(data.Keys))
}

func TestDataDtoContainsKnownHosts(t *testing.T) {
	// This test verifies that the DataDto structure contains the KnownHosts field
	// and that it's properly handled in JSON marshaling/unmarshaling

	// Create a DataDto with known_hosts data
	knownHostsData := []byte("github.com ssh-rsa AAAAB3NzaC1yc2EAAAA...")
	dataDto := dto.DataDto{
		ID:       uuid.New(),
		Username: "test",
		Keys: []dto.KeyDto{
			{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Filename: "id_rsa",
				Data:     []byte("key-data"),
			},
		},
		KnownHosts: knownHostsData,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(dataDto)
	assert.NoError(t, err)

	// Unmarshal back to struct
	var unmarshaledDto dto.DataDto
	err = json.Unmarshal(jsonData, &unmarshaledDto)
	assert.NoError(t, err)

	// Verify known_hosts field is preserved
	assert.Equal(t, knownHostsData, unmarshaledDto.KnownHosts)
}

func TestDeleteKey(t *testing.T) {
	// Arrange
	key := dto.KeyDto{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Filename: "test",
	}
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	err := retrieval.DeleteKey(profile, key)
	// Assert
	assert.Nil(t, err)
}
