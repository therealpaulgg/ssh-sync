package retrieval

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func TestDownloadData(t *testing.T) {
	// Arrange
	client, masterKey := newTestClient(t)
	plaintext := []byte("test key contents")
	encryptedKey, err := utils.EncryptWithMasterKey(plaintext, masterKey)
	require.NoError(t, err)

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
	defer server.Close()
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	data, err := client.GetUserData(profile)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, len(data.Keys))
	assert.Equal(t, plaintext, data.Keys[0].Data)
}

func TestDeleteKey(t *testing.T) {
	// Arrange
	client, _ := newTestClient(t)
	key := dto.KeyDto{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Filename: "test",
	}
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	url, _ := url.Parse(server.URL)
	profile.ServerUrl = *url
	// Act
	err := client.DeleteKey(profile, key)
	// Assert
	assert.Nil(t, err)
}

func newTestClient(t *testing.T) (Client, []byte) {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return Client{
		GetToken:          func() (string, error) { return "test-token", nil },
		RetrieveMasterKey: func() ([]byte, error) { return key, nil },
	}, key
}
