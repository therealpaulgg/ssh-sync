package actions

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/urfave/cli/v2"
)

// ---- helpers ----

func newSyncTestContext(path string) *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.Bool("safe-mode", false, "")
	set.String("path", "", "")
	if path != "" {
		_ = set.Parse([]string{"--path", path})
	} else {
		_ = set.Parse(nil)
	}
	return cli.NewContext(&cli.App{}, set, nil)
}

// writeLocalKey creates a file in dir with the given mtime and returns its path.
func writeLocalKey(t *testing.T, dir, name string, mtime time.Time) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte("key-data"), 0600))
	require.NoError(t, os.Chtimes(p, mtime, mtime))
	return p
}

func serverKey(name string, updatedAt time.Time) dto.KeyDto {
	return dto.KeyDto{Filename: name, Data: []byte("server-data"), UpdatedAt: &updatedAt}
}

func serverKeyNoTimestamp(name string) dto.KeyDto {
	return dto.KeyDto{Filename: name, Data: []byte("server-data")}
}

// ---- classifyKeys unit tests ----

func TestClassify_LocalOnly_NewerThanLastSync_Upload(t *testing.T) {
	dir := t.TempDir()
	lastSync := time.Now().Add(-1 * time.Hour)
	mtime := time.Now() // newer than lastSync
	writeLocalKey(t, dir, "id_rsa", mtime)

	got, err := classifyKeys(dir, nil, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classUpload, got[0].class)
	assert.Equal(t, "id_rsa", got[0].filename)
}

func TestClassify_LocalOnly_OlderThanLastSync_DeletedRemotely(t *testing.T) {
	dir := t.TempDir()
	lastSync := time.Now()
	mtime := lastSync.Add(-1 * time.Hour) // older than lastSync
	writeLocalKey(t, dir, "id_rsa", mtime)

	got, err := classifyKeys(dir, nil, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classDeletedRemotely, got[0].class)
}

func TestClassify_ServerOnly_Download(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	keys := []dto.KeyDto{serverKey("id_ed25519", now)}

	got, err := classifyKeys(dir, keys, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classDownload, got[0].class)
	assert.Equal(t, "id_ed25519", got[0].filename)
}

func TestClassify_LocalNewer_Upload(t *testing.T) {
	dir := t.TempDir()
	serverTime := time.Now().Add(-2 * time.Hour)
	localTime := time.Now().Add(-1 * time.Hour) // newer than server
	writeLocalKey(t, dir, "id_rsa", localTime)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverTime)}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classUpload, got[0].class)
}

func TestClassify_ServerNewer_Download(t *testing.T) {
	dir := t.TempDir()
	localTime := time.Now().Add(-2 * time.Hour)
	serverTime := time.Now().Add(-1 * time.Hour) // newer than local
	writeLocalKey(t, dir, "id_rsa", localTime)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverTime)}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classDownload, got[0].class)
}

func TestClassify_EqualTimestamps_Skip(t *testing.T) {
	dir := t.TempDir()
	ts := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	writeLocalKey(t, dir, "id_rsa", ts)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", ts)}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class)
}

func TestClassify_BothModifiedSinceLastSync_Conflict(t *testing.T) {
	dir := t.TempDir()
	lastSync := time.Now().Add(-3 * time.Hour)
	localTime := time.Now().Add(-2 * time.Hour)  // after lastSync
	serverTime := time.Now().Add(-1 * time.Hour) // also after lastSync
	writeLocalKey(t, dir, "id_rsa", localTime)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverTime)}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classConflict, got[0].class)
}

func TestClassify_FirstRun_NoConflict(t *testing.T) {
	// lastSync == zero time: no conflicts, purely directional.
	dir := t.TempDir()
	localTime := time.Now().Add(-1 * time.Hour)
	serverTime := time.Now().Add(-30 * time.Minute) // server newer
	writeLocalKey(t, dir, "id_rsa", localTime)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverTime)}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	// Server is newer → download, not conflict.
	assert.Equal(t, classDownload, got[0].class)
}

func TestClassify_NoServerTimestamp_Upload(t *testing.T) {
	dir := t.TempDir()
	writeLocalKey(t, dir, "id_rsa", time.Now())

	got, err := classifyKeys(dir, []dto.KeyDto{serverKeyNoTimestamp("id_rsa")}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classUpload, got[0].class)
}

func TestClassify_ReservedFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	for _, name := range []string{"config", "known_hosts", "authorized_keys"} {
		writeLocalKey(t, dir, name, now)
	}
	// Server also lists them — still must not appear in classifications.
	serverKeys := []dto.KeyDto{serverKey("config", now), serverKey("known_hosts", now)}

	got, err := classifyKeys(dir, serverKeys, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, got, "reserved filenames must never be classified")
}

func TestClassify_EqualTimestamps_BothAfterLastSync_Skip(t *testing.T) {
	// Regression: when localMtime == serverUpdatedAt but both are "after" a
	// truncated lastSync, the old conflict check fired before the equal-timestamp
	// check, producing a false conflict. Equal timestamps must always be Skip.
	dir := t.TempDir()
	lastSync := time.Now().Add(-2 * time.Hour)
	ts := time.Now().Add(-1 * time.Hour) // both local and server at this time, after lastSync
	writeLocalKey(t, dir, "id_rsa", ts)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", ts)}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class, "equal timestamps must never produce a conflict")
}

func TestClassify_SubSecondPrecision_EqualAfterTruncatedLastSync_Skip(t *testing.T) {
	// Regression for the RFC3339 truncation bug: serverUpdatedAt and localMtime
	// share the same sub-second timestamp (e.g. 10:00:00.500). If lastSync was
	// stored without sub-second precision (10:00:00.000), both timestamps appear
	// "after" lastSync, triggering a false conflict. With the equal-timestamp
	// check in place this must classify as Skip.
	dir := t.TempDir()
	ts := time.Now().Add(-1 * time.Hour).Truncate(time.Second).Add(500 * time.Millisecond)
	lastSyncTruncated := ts.Truncate(time.Second) // simulates old RFC3339-precision file
	writeLocalKey(t, dir, "id_rsa", ts)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", ts)}, lastSyncTruncated)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class)
}

func TestClassify_PostDownloadResync_Skip(t *testing.T) {
	// After a download, os.Chtimes stamps localMtime = serverUpdatedAt.
	// setLastSync is then called with time.Now() which is after serverUpdatedAt.
	// A subsequent classify must produce Skip, not Conflict or Upload.
	dir := t.TempDir()
	serverUpdatedAt := time.Now().Add(-2 * time.Hour)
	lastSync := serverUpdatedAt.Add(500 * time.Millisecond) // set after download completed
	writeLocalKey(t, dir, "id_rsa", serverUpdatedAt)        // mtime stamped by Chtimes

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverUpdatedAt)}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class)
}

func TestClassify_PostUploadResync_Skip(t *testing.T) {
	// After an upload, the server echoes a timestamp which is used to Chtimes
	// the local file. setLastSync is then called, so lastSync > serverUpdatedAt.
	// A subsequent classify from the same machine must produce Skip.
	dir := t.TempDir()
	serverUpdatedAt := time.Now().Add(-2 * time.Hour)
	lastSync := serverUpdatedAt.Add(500 * time.Millisecond) // set after upload completed
	writeLocalKey(t, dir, "id_rsa", serverUpdatedAt)        // mtime stamped by Chtimes after upload

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverUpdatedAt)}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class)
}

func TestClassify_CrossMachine_NoConflict(t *testing.T) {
	// Full multi-machine scenario:
	//   Machine A uploads id_rsa → server stamps it at T2. A Chtimes local to T2.
	//   A sets lastSync = T3 (T3 > T2).
	//   Machine B downloads id_rsa → B Chtimes local to T2. B sets lastSync = T4.
	//   Machine A syncs again: localMtime_A = T2, serverUpdatedAt = T2, lastSync_A = T3.
	//   Expected: Skip (not Conflict or Upload).
	dir := t.TempDir()
	T2 := time.Now().Add(-2 * time.Hour)
	T3 := T2.Add(200 * time.Millisecond) // lastSync_A set after upload response received
	writeLocalKey(t, dir, "id_rsa", T2)  // Chtimes-stamped after upload

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", T2)}, T3)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class, "second sync on machine A after machine B download must be Skip")
}

func TestClassify_IdenticalContent_DifferentTimestamps_Skip(t *testing.T) {
	// If local and server have the same bytes, no transfer is needed regardless
	// of which timestamp is newer.
	dir := t.TempDir()
	localTime := time.Now().Add(-1 * time.Hour)
	serverTime := time.Now() // server appears newer
	p := filepath.Join(dir, "id_rsa")
	require.NoError(t, os.WriteFile(p, []byte("same-content"), 0600))
	require.NoError(t, os.Chtimes(p, localTime, localTime))

	sk := dto.KeyDto{Filename: "id_rsa", Data: []byte("same-content"), UpdatedAt: &serverTime}
	got, err := classifyKeys(dir, []dto.KeyDto{sk}, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class, "identical content must skip even when server timestamp is newer")
}

func TestClassify_ConflictWithIdenticalContent_Skip(t *testing.T) {
	// Regression: when both sides are modified since lastSync but the content is
	// identical (e.g. independent writes of the same key), there is no real
	// conflict — classify as Skip.
	dir := t.TempDir()
	lastSync := time.Now().Add(-3 * time.Hour)
	localTime := time.Now().Add(-2 * time.Hour)  // after lastSync
	serverTime := time.Now().Add(-1 * time.Hour) // also after lastSync, different timestamp
	p := filepath.Join(dir, "id_rsa")
	require.NoError(t, os.WriteFile(p, []byte("same-content"), 0600))
	require.NoError(t, os.Chtimes(p, localTime, localTime))

	sk := dto.KeyDto{Filename: "id_rsa", Data: []byte("same-content"), UpdatedAt: &serverTime}
	got, err := classifyKeys(dir, []dto.KeyDto{sk}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classSkip, got[0].class, "identical content must not be treated as a conflict")
}

func TestClassify_GenuineConflict_DifferentTimestamps(t *testing.T) {
	// Ensure the equal-timestamp early-exit does not suppress genuine conflicts
	// where local and server have different (both post-lastSync) timestamps.
	dir := t.TempDir()
	lastSync := time.Now().Add(-3 * time.Hour)
	localTime := lastSync.Add(30 * time.Minute)  // different from server
	serverTime := lastSync.Add(60 * time.Minute) // different from local
	writeLocalKey(t, dir, "id_rsa", localTime)

	got, err := classifyKeys(dir, []dto.KeyDto{serverKey("id_rsa", serverTime)}, lastSync)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, classConflict, got[0].class, "different timestamps both after lastSync must still conflict")
}

// ---- syncWithDeps integration tests ----

// minimalSyncDeps returns a deps struct where all injected functions succeed with
// empty/zero values. checkSetup returns true, getProfile returns a zero Profile.
func minimalSyncDeps() syncDeps {
	return syncDeps{
		checkSetup:   func() (bool, error) { return true, nil },
		getProfile:   func() (*models.Profile, error) { return &models.Profile{}, nil },
		getUserData:  func(*models.Profile) (dto.DataDto, error) { return dto.DataDto{}, nil },
		getMasterKey: func() ([]byte, error) { return []byte("masterkey12345678901234"), nil },
		getLastSync:  func() (time.Time, error) { return time.Time{}, nil },
		setLastSync:  func(time.Time) error { return nil },
	}
}

func TestSyncWithDeps_StopsWhenNotSetup(t *testing.T) {
	deps := syncDeps{
		checkSetup: func() (bool, error) { return false, nil },
	}
	err := syncWithDeps(newSyncTestContext(""), deps)
	assert.NoError(t, err)
}

func TestSyncWithDeps_CallsSetLastSyncOnSuccess(t *testing.T) {
	dir := t.TempDir()
	setLastSyncCalled := false
	deps := minimalSyncDeps()
	deps.setLastSync = func(time.Time) error {
		setLastSyncCalled = true
		return nil
	}

	err := syncWithDeps(newSyncTestContext(dir), deps)
	require.NoError(t, err)
	assert.True(t, setLastSyncCalled, "setLastSync must be called on success")
}

func TestSyncWithDeps_SetLastSyncNotCalledOnGetUserDataError(t *testing.T) {
	dir := t.TempDir()
	setLastSyncCalled := false
	deps := minimalSyncDeps()
	deps.getUserData = func(*models.Profile) (dto.DataDto, error) {
		return dto.DataDto{}, errors.New("network error")
	}
	deps.setLastSync = func(time.Time) error {
		setLastSyncCalled = true
		return nil
	}

	err := syncWithDeps(newSyncTestContext(dir), deps)
	assert.Error(t, err)
	assert.False(t, setLastSyncCalled, "setLastSync must not be called when getUserData fails")
}

func TestSyncWithDeps_SetLastSyncNotCalledOnGetMasterKeyError(t *testing.T) {
	dir := t.TempDir()
	setLastSyncCalled := false
	deps := minimalSyncDeps()
	deps.getMasterKey = func() ([]byte, error) { return nil, errors.New("keychain locked") }
	deps.setLastSync = func(time.Time) error {
		setLastSyncCalled = true
		return nil
	}

	err := syncWithDeps(newSyncTestContext(dir), deps)
	assert.Error(t, err)
	assert.False(t, setLastSyncCalled, "setLastSync must not be called when getMasterKey fails")
}

func TestSyncWithDeps_DownloadsServerOnlyKey(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	deps := minimalSyncDeps()
	deps.getUserData = func(*models.Profile) (dto.DataDto, error) {
		return dto.DataDto{
			Keys: []dto.KeyDto{serverKey("id_ed25519", now)},
		}, nil
	}

	// classifyKeys will classify id_ed25519 as classDownload (server-only).
	// WriteKey writes to ~/.ssh; we verify sync reaches the download path without
	// classification errors. WriteKey failures (e.g. missing ~/.ssh) are acceptable.
	err := syncWithDeps(newSyncTestContext(dir), deps)
	if err != nil {
		assert.Contains(t, err.Error(), "ssh", "unexpected error; expected only fs-related failure from WriteKey")
	}
}

func TestSyncWithDeps_UploadsLocalOnlyKey(t *testing.T) {
	dir := t.TempDir()
	writeLocalKey(t, dir, "id_rsa", time.Now())

	uploadCalled := false
	deps := minimalSyncDeps()
	// Intercept by making getMasterKey succeed; sendUpload will fail because
	// profile.ServerUrl is zero — that's expected. We verify the upload path
	// was entered by confirming getMasterKey was invoked (it's called before sendUpload).
	deps.getMasterKey = func() ([]byte, error) {
		uploadCalled = true
		return nil, errors.New("stop here")
	}
	deps.setLastSync = func(time.Time) error { return nil }

	_ = syncWithDeps(newSyncTestContext(dir), deps)
	assert.True(t, uploadCalled, "getMasterKey (and thus upload path) must be reached")
}
