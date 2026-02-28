package retrieval

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func newMultipart(t *testing.T) (*multipart.Writer, bytes.Buffer) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.Close()
	return mw, body
}

func TestUploadData(t *testing.T) {
	// Arrange
	client, _ := newTestClient(t)
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/data", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]dto.KeyDto{})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	profile.ServerUrl = *u
	mw, body := newMultipart(t)
	// Act
	err := client.UploadData(profile, t.TempDir(), mw, body)
	// Assert
	assert.Nil(t, err)
}

func TestUploadDataServerError(t *testing.T) {
	// Arrange
	client, _ := newTestClient(t)
	profile := &models.Profile{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	profile.ServerUrl = *u
	mw, body := newMultipart(t)
	// Act
	err := client.UploadData(profile, t.TempDir(), mw, body)
	// Assert
	assert.NotNil(t, err)
}

func TestUploadDataSetsFileTimestamps(t *testing.T) {
	// Arrange
	client, _ := newTestClient(t)
	profile := &models.Profile{}
	tmpDir := t.TempDir()
	filename := "id_rsa"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, filename), []byte("key data"), 0600))

	updatedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]dto.KeyDto{
			{ID: uuid.New(), Filename: filename, UpdatedAt: &updatedAt},
		})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	profile.ServerUrl = *u
	mw, body := newMultipart(t)
	// Act
	err := client.UploadData(profile, tmpDir, mw, body)
	// Assert
	assert.Nil(t, err)
	fi, err := os.Stat(filepath.Join(tmpDir, filename))
	require.NoError(t, err)
	assert.Equal(t, updatedAt.Truncate(time.Second), fi.ModTime().UTC().Truncate(time.Second))
}

func TestUploadDataIgnoresMissingFileOnChtimes(t *testing.T) {
	// Arrange: server returns a key whose filename does not exist locally.
	// UploadData silently ignores Chtimes failures, so no error should be returned.
	client, _ := newTestClient(t)
	profile := &models.Profile{}
	updatedAt := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]dto.KeyDto{
			{ID: uuid.New(), Filename: "nonexistent", UpdatedAt: &updatedAt},
		})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	profile.ServerUrl = *u
	mw, body := newMultipart(t)
	// Act
	err := client.UploadData(profile, t.TempDir(), mw, body)
	// Assert
	assert.Nil(t, err)
}

func TestUploadDataSendsAuthHeader(t *testing.T) {
	// Arrange
	client, _ := newTestClient(t)
	profile := &models.Profile{}
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]dto.KeyDto{})
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	profile.ServerUrl = *u
	mw, body := newMultipart(t)
	// Act
	err := client.UploadData(profile, t.TempDir(), mw, body)
	// Assert
	assert.Nil(t, err)
	assert.Equal(t, "Bearer test-token", receivedAuth)
}

func newTestClient(t *testing.T) (RetrievalClient, []byte) {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return RetrievalClient{
		GetToken:          func() (string, error) { return "test-token", nil },
		RetrieveMasterKey: func() ([]byte, error) { return key, nil },
	}, key
}
