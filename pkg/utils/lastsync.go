package utils

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

// GetLastSync reads the last successful sync timestamp from ~/.ssh-sync/last_sync.
// Returns a zero time.Time (no error) when the file does not yet exist.
// The file is written in RFC3339Nano format; RFC3339 is accepted as a fallback for old files.
func GetLastSync() (time.Time, error) {
	u, err := user.Current()
	if err != nil {
		return time.Time{}, err
	}
	return getLastSyncFromFile(filepath.Join(u.HomeDir, ".ssh-sync", "last_sync"))
}

// SetLastSync writes the given timestamp to ~/.ssh-sync/last_sync in RFC3339Nano format.
func SetLastSync(t time.Time) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	return setLastSyncToFile(filepath.Join(u.HomeDir, ".ssh-sync", "last_sync"), t)
}

func getLastSyncFromFile(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, string(data))
	if err != nil {
		t, err = time.Parse(time.RFC3339, string(data)) // fallback for files written before RFC3339Nano
	}
	return t, err
}

func setLastSyncToFile(path string, t time.Time) error {
	return os.WriteFile(path, []byte(t.Format(time.RFC3339Nano)), 0600)
}
