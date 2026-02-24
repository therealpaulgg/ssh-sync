package retrieval

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func TestDownloadData(t *testing.T) {
	// Arrange
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		t.Skipf("ssh-sync not configured, skipping: %v", err)
	}
	plaintext := []byte("test key contents")
	encryptedKey, err := utils.EncryptWithMasterKey(plaintext, masterKey)
	if err != nil {
		t.Fatalf("encrypting test key: %v", err)
	}
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(dto.DataDto{
			ID:       uuid.New(),
			Username: "test",
			Keys: []dto.KeyDto{
				{
					ID:       uuid.New(),
					UserID:   uuid.New(),
					Filename: "test",
					Data:     encryptedKey,
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
		})
	}))
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	data, err := GetUserData(profile)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, len(data.Keys))
	assert.Equal(t, plaintext, data.Keys[0].Data)
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
	err := DeleteKey(profile, key)
	// Assert
	assert.Nil(t, err)
}
