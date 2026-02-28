package actions

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/urfave/cli/v2"
)

func TestSyncChoosesDownloadFirst(t *testing.T) {
	steps := []string{}
	deps := syncDeps{
		checkSetup: func() (bool, error) { return true, nil },
		getProfile: func() (*models.Profile, error) { return &models.Profile{}, nil },
		getData:    func(*models.Profile) (dto.DataDto, error) { return dto.DataDto{}, nil },
		upload: func(*cli.Context) error {
			steps = append(steps, "upload")
			return nil
		},
		download: func(*cli.Context) error {
			steps = append(steps, "download")
			return nil
		},
		decide: func(*cli.Context, dto.DataDto) bool { return true },
	}

	err := syncWithDeps(newSyncTestContext(nil), deps)
	assert.NoError(t, err)
	assert.Equal(t, []string{"download", "upload"}, steps)
}

func TestSyncChoosesUploadFirst(t *testing.T) {
	steps := []string{}
	deps := syncDeps{
		checkSetup: func() (bool, error) { return true, nil },
		getProfile: func() (*models.Profile, error) { return &models.Profile{}, nil },
		getData:    func(*models.Profile) (dto.DataDto, error) { return dto.DataDto{}, nil },
		upload: func(*cli.Context) error {
			steps = append(steps, "upload")
			return nil
		},
		download: func(*cli.Context) error {
			steps = append(steps, "download")
			return nil
		},
		decide: func(*cli.Context, dto.DataDto) bool { return false },
	}

	err := syncWithDeps(newSyncTestContext(nil), deps)
	assert.NoError(t, err)
	assert.Equal(t, []string{"upload", "download"}, steps)
}

func TestShouldDownloadFirstWhenServerNewer(t *testing.T) {
	tmp := t.TempDir()
	localKey := filepath.Join(tmp, "id_ed25519")
	now := time.Now()
	if err := os.WriteFile(localKey, []byte("old"), 0600); err != nil {
		t.Fatalf("write local key: %v", err)
	}
	future := now.Add(10 * time.Minute)

	result := shouldDownloadFirst(newSyncTestContext([]string{"--path", tmp}), dto.DataDto{
		Keys: []dto.KeyDto{
			{Filename: "id_ed25519", UpdatedAt: &future},
		},
	})
	assert.True(t, result)
}

func TestShouldDownloadFirstWhenLocalMissingKey(t *testing.T) {
	tmp := t.TempDir()
	future := time.Now()

	result := shouldDownloadFirst(newSyncTestContext([]string{"--path", tmp}), dto.DataDto{
		Keys: []dto.KeyDto{
			{Filename: "id_rsa", UpdatedAt: &future},
		},
	})
	assert.True(t, result)
}

func TestShouldDownloadFirstWhenLocalNewer(t *testing.T) {
	tmp := t.TempDir()
	localKey := filepath.Join(tmp, "id_ed25519")
	now := time.Now()
	past := now.Add(-10 * time.Minute)
	if err := os.WriteFile(localKey, []byte("new"), 0600); err != nil {
		t.Fatalf("write local key: %v", err)
	}
	if err := os.Chtimes(localKey, now, now); err != nil {
		t.Fatalf("set times: %v", err)
	}

	result := shouldDownloadFirst(newSyncTestContext([]string{"--path", tmp}), dto.DataDto{
		Keys: []dto.KeyDto{
			{Filename: "id_ed25519", UpdatedAt: &past},
		},
	})
	assert.False(t, result)
}

func newSyncTestContext(args []string) *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.Bool("safe-mode", false, "")
	set.String("path", "", "")
	_ = set.Parse(args)
	return cli.NewContext(&cli.App{}, set, nil)
}
