package retrieval

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
	data, err := GetUserData(profile)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, len(data.Keys))
}
