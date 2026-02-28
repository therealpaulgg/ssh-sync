package actions

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
)

// testMasterKey is a valid 32-byte AES-256 key used in all sync tests.
var testMasterKey = bytes.Repeat([]byte{0x42}, 32)

// writeLocalKey writes content to dir/name and sets its mtime to t.
func writeLocalKey(t *testing.T, dir, name string, content []byte, mtime time.Time) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, content, 0600))
	require.NoError(t, os.Chtimes(p, mtime, mtime))
	return p
}

// serverKey builds a KeyDto with the given filename, (decrypted) data, and UpdatedAt.
func serverKey(filename string, data []byte, updatedAt *time.Time) dto.KeyDto {
	return dto.KeyDto{Filename: filename, Data: data, UpdatedAt: updatedAt}
}

// --- Verification step 1: server-newer key is downloaded silently ---

func TestBuildSyncDecisions_ServerNewer_Download(t *testing.T) {
	dir := t.TempDir()
	localTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	serverTime := localTime.Add(time.Hour)

	writeLocalKey(t, dir, "id_rsa", []byte("old-local"), localTime)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", []byte("new-server"), &serverTime)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "download", decisions[0].action)
	assert.Equal(t, "id_rsa", decisions[0].filename)
	assert.Empty(t, toUpload)
}

// After applyDownloads the file on disk matches the server content and mtime,
// so a second buildSyncDecisions call produces a "skip" decision.

func TestBuildSyncDecisions_AlreadyInSync_Skip(t *testing.T) {
	dir := t.TempDir()
	syncTime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	content := []byte("synced-content")

	writeLocalKey(t, dir, "id_ed25519", content, syncTime)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_ed25519", content, &syncTime)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "skip", decisions[0].action)
	assert.Empty(t, toUpload)
}

// --- Verification step 2: new local key is uploaded silently ---

func TestBuildSyncDecisions_LocalOnly_Upload(t *testing.T) {
	dir := t.TempDir()
	writeLocalKey(t, dir, "id_ecdsa", []byte("local-only"), time.Now())

	decisions, toUpload, err := buildSyncDecisions(dir, nil, testMasterKey)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "upload", decisions[0].action)
	assert.Equal(t, "id_ecdsa", decisions[0].filename)
	require.Len(t, toUpload, 1)
	assert.Equal(t, "id_ecdsa", toUpload[0].name)
	assert.NotEmpty(t, toUpload[0].encData, "encrypted data must be populated")
}

func TestBuildSyncDecisions_LocalNewer_Upload(t *testing.T) {
	dir := t.TempDir()
	serverTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	localTime := serverTime.Add(time.Hour)

	writeLocalKey(t, dir, "id_rsa", []byte("local-newer"), localTime)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", []byte("server-older"), &serverTime)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "upload", decisions[0].action)
	require.Len(t, toUpload, 1)
	assert.Equal(t, "id_rsa", toUpload[0].name)
}

// --- Equal timestamp with different content: server wins (no prompt) ---

func TestBuildSyncDecisions_EqualTimestamp_DifferentContent_ServerWins(t *testing.T) {
	dir := t.TempDir()
	ts := time.Now().Add(-time.Hour).Truncate(time.Second)

	writeLocalKey(t, dir, "id_rsa", []byte("local-content"), ts)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", []byte("server-content"), &ts)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "download", decisions[0].action)
	assert.Empty(t, toUpload)
}

// Sub-second server timestamps must be truncated before comparison so that a
// filesystem that only stores second-level mtime (e.g. HFS+) doesn't cause
// infinite re-downloads.

func TestBuildSyncDecisions_SubSecondServerTimestamp_Skipped(t *testing.T) {
	dir := t.TempDir()
	base := time.Now().Add(-time.Hour).Truncate(time.Second)
	serverTime := base.Add(456 * time.Microsecond) // sub-second component
	content := []byte("same-content")

	// local mtime is the second-truncated value, as HFS+ would store it
	writeLocalKey(t, dir, "id_rsa", content, base)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", content, &serverTime)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "skip", decisions[0].action, "sub-second difference with identical content should skip, not re-download")
	assert.Empty(t, toUpload)
}

// --- No server timestamp: server wins if content differs ---

func TestBuildSyncDecisions_NilUpdatedAt_SameContent_Skip(t *testing.T) {
	dir := t.TempDir()
	content := []byte("same")
	writeLocalKey(t, dir, "id_rsa", content, time.Now())

	decisions, _, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", content, nil)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "skip", decisions[0].action)
}

func TestBuildSyncDecisions_NilUpdatedAt_DifferentContent_ServerWins(t *testing.T) {
	dir := t.TempDir()
	writeLocalKey(t, dir, "id_rsa", []byte("local"), time.Now())

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", []byte("server"), nil)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "download", decisions[0].action)
	assert.Empty(t, toUpload)
}

// --- Verification step 4: key present on server but not locally is downloaded ---

func TestBuildSyncDecisions_ServerKeyNotLocal_Download(t *testing.T) {
	dir := t.TempDir() // empty — no local keys
	serverTime := time.Now().Add(-time.Hour).Truncate(time.Second)

	decisions, toUpload, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{serverKey("id_rsa", []byte("from-another-machine"), &serverTime)},
		testMasterKey,
	)

	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "download", decisions[0].action)
	assert.Equal(t, "id_rsa", decisions[0].filename)
	assert.Empty(t, toUpload)
}

// Reserved filenames on the server must never be downloaded as keys.

func TestBuildSyncDecisions_ServerReservedFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	ts := time.Now().Truncate(time.Second)

	decisions, _, err := buildSyncDecisions(
		dir,
		[]dto.KeyDto{
			serverKey("config", []byte("Host *"), &ts),
			serverKey("known_hosts", []byte("github.com ssh-ed25519 AAAA"), &ts),
			serverKey("authorized_keys", []byte("ssh-rsa AAAA"), &ts),
		},
		testMasterKey,
	)

	require.NoError(t, err)
	assert.Empty(t, decisions, "reserved filenames must be excluded from decisions")
}

// Local reserved / skipped filenames must also be excluded from decisions.

func TestBuildSyncDecisions_LocalSkippedFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"authorized_keys", "known_hosts", "config"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0600))
	}

	decisions, toUpload, err := buildSyncDecisions(dir, nil, testMasterKey)

	require.NoError(t, err)
	assert.Empty(t, decisions)
	assert.Empty(t, toUpload)
}

// --- Verification step 5: safe-mode — applyDownloads writes to the given directory ---

func TestApplyDownloads_WritesFileAndTimestamp(t *testing.T) {
	dir := t.TempDir()
	serverTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	content := []byte("downloaded-key-data")

	decisions := []syncDecision{{filename: "id_rsa", action: "download"}}
	serverMap := map[string]dto.KeyDto{
		"id_rsa": serverKey("id_rsa", content, &serverTime),
	}

	require.NoError(t, applyDownloads(decisions, serverMap, dir))

	written, err := os.ReadFile(filepath.Join(dir, "id_rsa"))
	require.NoError(t, err)
	assert.Equal(t, content, written)

	info, err := os.Stat(filepath.Join(dir, "id_rsa"))
	require.NoError(t, err)
	assert.Equal(t, serverTime, info.ModTime(), "mtime must match server UpdatedAt")
}

func TestApplyDownloads_SafeMode_WritesToAlternateDir(t *testing.T) {
	sshDir := t.TempDir()  // simulates ~/.ssh  — must stay empty
	safeDir := t.TempDir() // simulates ~/.ssh-sync-data
	serverTime := time.Now().Add(-time.Hour).Truncate(time.Second)

	decisions := []syncDecision{{filename: "id_ed25519", action: "download"}}
	serverMap := map[string]dto.KeyDto{
		"id_ed25519": serverKey("id_ed25519", []byte("key"), &serverTime),
	}

	require.NoError(t, applyDownloads(decisions, serverMap, safeDir))

	// File must appear in the safe directory.
	_, err := os.Stat(filepath.Join(safeDir, "id_ed25519"))
	require.NoError(t, err, "file should be in safe-mode directory")

	// The original ssh directory must be untouched.
	entries, err := os.ReadDir(sshDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "~/.ssh must not be modified in safe mode")
}

// Upload decisions must not produce any writes — only the toUpload slice is populated.

func TestBuildSyncDecisions_UploadDoesNotWriteLocally(t *testing.T) {
	dir := t.TempDir()
	writeLocalKey(t, dir, "id_rsa", []byte("local-only"), time.Now())

	decisions, toUpload, err := buildSyncDecisions(dir, nil, testMasterKey)
	require.NoError(t, err)

	require.Len(t, decisions, 1)
	assert.Equal(t, "upload", decisions[0].action)
	require.Len(t, toUpload, 1)

	// The directory must contain exactly the file we put there — nothing extra written.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}
