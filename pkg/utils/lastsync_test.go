package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLastSync_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last_sync")
	got, err := getLastSyncFromFile(path)
	require.NoError(t, err)
	assert.True(t, got.IsZero(), "missing file should return zero time")
}

func TestSetGetLastSync_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last_sync")
	// Round(0) strips the monotonic clock reading that time.Parse doesn't restore.
	now := time.Now().UTC().Round(0)
	require.NoError(t, setLastSyncToFile(path, now))
	got, err := getLastSyncFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, now, got)
}

func TestSetGetLastSync_SubSecondPrecision(t *testing.T) {
	// Regression: RFC3339 truncated to seconds, causing false conflicts when
	// serverUpdatedAt had sub-second precision and lastSync was stored without it.
	// RFC3339Nano must preserve sub-second digits.
	dir := t.TempDir()
	path := filepath.Join(dir, "last_sync")
	ts := time.Date(2024, 1, 15, 10, 30, 0, 500_000_000, time.UTC) // exactly .5 s
	require.NoError(t, setLastSyncToFile(path, ts))
	got, err := getLastSyncFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, ts, got, "sub-second precision must survive a round-trip")
}

func TestGetLastSync_RFC3339FallbackCompat(t *testing.T) {
	// Old last_sync files written in RFC3339 (no sub-second digits) must still parse.
	dir := t.TempDir()
	path := filepath.Join(dir, "last_sync")
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	require.NoError(t, os.WriteFile(path, []byte(ts.Format(time.RFC3339)), 0600))
	got, err := getLastSyncFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, ts, got, "old RFC3339 files must still be readable")
}
