package actions

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

type keyClass int

const (
	classUpload keyClass = iota
	classDownload
	classSkip
	classConflict
	classDeletedRemotely
)

type keyClassification struct {
	filename  string
	class     keyClass
	localPath string
	serverKey *dto.KeyDto
}

// classifyKeys compares the local SSH directory against the server key list and
// assigns each key a sync direction (upload/download/skip/conflict/deletedRemotely).
func classifyKeys(localDir string, serverKeys []dto.KeyDto, lastSync time.Time) ([]keyClassification, error) {
	serverByFilename := make(map[string]*dto.KeyDto, len(serverKeys))
	for i := range serverKeys {
		serverByFilename[serverKeys[i].Filename] = &serverKeys[i]
	}

	localEntries, err := os.ReadDir(localDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var result []keyClassification
	seenLocal := make(map[string]struct{})

	for _, entry := range localEntries {
		if entry.IsDir() || isReservedFilename(entry.Name()) {
			continue
		}
		seenLocal[entry.Name()] = struct{}{}

		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		localMtime := info.ModTime()
		localPath := filepath.Join(localDir, entry.Name())

		serverKey, onServer := serverByFilename[entry.Name()]
		if !onServer {
			// local-only
			if !lastSync.IsZero() && !localMtime.After(lastSync) {
				// Existed before last sync and not touched locally → deleted remotely.
				result = append(result, keyClassification{
					filename:  entry.Name(),
					class:     classDeletedRemotely,
					localPath: localPath,
				})
			} else {
				// New or modified since last sync → upload.
				result = append(result, keyClassification{
					filename:  entry.Name(),
					class:     classUpload,
					localPath: localPath,
				})
			}
			continue
		}

		// Key exists on both sides.
		if serverKey.UpdatedAt == nil {
			// No server timestamp — upload to be safe.
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classUpload,
				localPath: localPath,
				serverKey: serverKey,
			})
			continue
		}

		serverUpdatedAt := *serverKey.UpdatedAt

		// Equal timestamps → definitionally in sync; skip conflict check entirely.
		if localMtime.Equal(serverUpdatedAt) {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classSkip,
				localPath: localPath,
				serverKey: serverKey,
			})
			continue
		}

		// Equal content → no transfer needed regardless of timestamp differences.
		localData, err := os.ReadFile(localPath)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(localData, serverKey.Data) {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classSkip,
				localPath: localPath,
				serverKey: serverKey,
			})
			continue
		}

		// Conflict: both sides modified since lastSync (only when lastSync is known).
		if !lastSync.IsZero() && localMtime.After(lastSync) && serverUpdatedAt.After(lastSync) {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classConflict,
				localPath: localPath,
				serverKey: serverKey,
			})
			continue
		}

		if localMtime.After(serverUpdatedAt) {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classUpload,
				localPath: localPath,
				serverKey: serverKey,
			})
		} else if serverUpdatedAt.After(localMtime) {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classDownload,
				localPath: localPath,
				serverKey: serverKey,
			})
		} else {
			result = append(result, keyClassification{
				filename:  entry.Name(),
				class:     classSkip,
				localPath: localPath,
				serverKey: serverKey,
			})
		}
	}

	// Server-only keys → download.
	for i := range serverKeys {
		sk := &serverKeys[i]
		if isReservedFilename(sk.Filename) {
			continue
		}
		if _, seen := seenLocal[sk.Filename]; !seen {
			result = append(result, keyClassification{
				filename:  sk.Filename,
				class:     classDownload,
				serverKey: sk,
			})
		}
	}

	return result, nil
}

// knownHostDtosToEntries converts server-side KnownHostDto values to the local
// KnownHostEntry model used by utils.WriteKnownHosts.
func knownHostDtosToEntries(dtos []dto.KnownHostDto) []models.KnownHostEntry {
	return lo.Map(dtos, func(kh dto.KnownHostDto, _ int) models.KnownHostEntry {
		return models.KnownHostEntry{
			HostPattern: kh.HostPattern,
			KeyType:     kh.KeyType,
			KeyData:     kh.KeyData,
			Marker:      kh.Marker,
		}
	})
}

type syncDeps struct {
	checkSetup   func() (bool, error)
	getProfile   func() (*models.Profile, error)
	getUserData  func(*models.Profile) (dto.DataDto, error)
	getMasterKey func() ([]byte, error)
	getLastSync  func() (time.Time, error)
	setLastSync  func(time.Time) error
}

func defaultSyncDeps() syncDeps {
	client := retrieval.NewRetrievalClient()
	return syncDeps{
		checkSetup:   utils.CheckIfSetup,
		getProfile:   utils.GetProfile,
		getUserData:  client.GetUserData,
		getMasterKey: utils.RetrieveMasterKey,
		getLastSync:  utils.GetLastSync,
		setLastSync:  utils.SetLastSync,
	}
}

func Sync(c *cli.Context) error {
	return syncWithDeps(c, defaultSyncDeps())
}

func syncWithDeps(c *cli.Context, deps syncDeps) error {
	setup, err := deps.checkSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}

	profile, err := deps.getProfile()
	if err != nil {
		return err
	}

	serverData, err := deps.getUserData(profile)
	if err != nil {
		return err
	}

	masterKey, err := deps.getMasterKey()
	if err != nil {
		return err
	}

	lastSync, err := deps.getLastSync()
	if err != nil {
		return err
	}

	isSafeMode := c.Bool("safe-mode")
	var sshDir string
	if isSafeMode {
		fmt.Println("Executing in safe mode (keys writing to .ssh-sync-data)")
		sshDir = ".ssh-sync-data"
	} else {
		sshDir = ".ssh"
	}

	// Resolve the local read directory.
	p := c.String("path")
	if p == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		p = filepath.Join(u.HomeDir, ".ssh")
	}

	classifications, err := classifyKeys(p, serverData.Keys, lastSync)
	if err != nil {
		return err
	}

	// Print summary and partition into action buckets in one pass.
	var toUpload, toDownload, toConflict []keyClassification
	for _, kc := range classifications {
		switch kc.class {
		case classUpload:
			fmt.Printf("  upload:           %s\n", kc.filename)
			toUpload = append(toUpload, kc)
		case classDownload:
			fmt.Printf("  download:         %s\n", kc.filename)
			toDownload = append(toDownload, kc)
		case classSkip:
			fmt.Printf("  skip (in sync):   %s\n", kc.filename)
		case classConflict:
			fmt.Printf("  conflict:         %s\n", kc.filename)
			toConflict = append(toConflict, kc)
		case classDeletedRemotely:
			fmt.Printf("  deleted remotely: %s (skipping local copy)\n", kc.filename)
		}
	}

	// --- Upload ---
	if len(toUpload) > 0 {
		keysMap := make(map[string][]byte, len(toUpload))
		for _, kc := range toUpload {
			fileBytes, err := os.ReadFile(kc.localPath)
			if err != nil {
				return fmt.Errorf("reading %s: %w", kc.filename, err)
			}
			keysMap[kc.filename] = fileBytes
		}

		// Parse local config and known_hosts to send alongside the keys.
		hosts, _ := utils.ParseConfig()
		var knownHosts []dto.KnownHostDto
		khPath := filepath.Join(p, "known_hosts")
		if khEntries, err := utils.ParseKnownHosts(khPath); err == nil {
			knownHosts = knownHostEntriesToDtos(khEntries)
		}

		token, err := utils.GetToken()
		if err != nil {
			return err
		}
		uploadedKeys, err := sendUpload(keysMap, hosts, knownHosts, masterKey, token, profile)
		if err != nil {
			return err
		}
		// Stamp local mtimes to match server timestamps so future syncs classify correctly.
		for _, key := range uploadedKeys {
			if key.UpdatedAt != nil {
				localPath := filepath.Join(p, key.Filename)
				_ = os.Chtimes(localPath, *key.UpdatedAt, *key.UpdatedAt)
			}
		}
	}

	// --- Download ---
	// Resolve the full write directory once so we can stamp mtimes after WriteKey.
	sshDirFull, err := utils.GetAndCreateSshDirectory(sshDir)
	if err != nil {
		return err
	}
	for _, kc := range toDownload {
		if err := utils.WriteKey(kc.serverKey.Data, kc.filename, sshDir); err != nil {
			return err
		}
		// Stamp local mtime to match the server timestamp so future syncs classify as skip.
		if kc.serverKey.UpdatedAt != nil {
			_ = os.Chtimes(filepath.Join(sshDirFull, kc.filename), *kc.serverKey.UpdatedAt, *kc.serverKey.UpdatedAt)
		}
	}

	// Write ssh config and known_hosts from server (mirrors download.go).
	if len(serverData.SshConfig) > 0 {
		if err := utils.WriteConfig(lo.Map(serverData.SshConfig, func(cfg dto.SshConfigDto, _ int) models.Host {
			return models.Host{
				Host:          cfg.Host,
				Values:        cfg.Values,
				IdentityFiles: cfg.IdentityFiles,
			}
		}), sshDir); err != nil {
			return err
		}
	}
	if len(serverData.KnownHosts) > 0 {
		if err := utils.WriteKnownHosts(knownHostDtosToEntries(serverData.KnownHosts), sshDir); err != nil {
			return err
		}
	}

	// --- Conflicts: prompt interactively via WriteKey ---
	for _, kc := range toConflict {
		fmt.Printf("Warning: conflict for %s — both local and server modified since last sync.\n", kc.filename)
		if err := utils.WriteKey(kc.serverKey.Data, kc.filename, sshDir); err != nil {
			return err
		}
	}

	if err := deps.setLastSync(time.Now()); err != nil {
		return err
	}

	fmt.Println("Sync complete.")
	return nil
}
