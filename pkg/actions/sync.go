package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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

type syncDecision struct {
	filename string
	action   string // "upload"|"download"|"skip"
}

type fileToUpload struct {
	name    string
	encData []byte
}

func Sync(c *cli.Context) error {
	setup, err := utils.CheckIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}

	if err := applyPendingRotation(profile); err != nil {
		return fmt.Errorf("applying pending key rotation: %w", err)
	}

	p := c.String("path")
	if p == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		p = filepath.Join(u.HomeDir, ".ssh")
	}

	isSafeMode := c.Bool("safe-mode")
	var downloadDir string
	if isSafeMode {
		fmt.Println("Executing in safe mode (downloads writing to .ssh-sync-data)")
		downloadDir = ".ssh-sync-data"
	} else {
		downloadDir = ".ssh"
	}

	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return err
	}

	client := retrieval.NewRetrievalClient()
	serverData, err := client.GetUserData(profile)
	if err != nil {
		return err
	}

	decisions, toUpload, err := buildSyncDecisions(p, serverData.Keys, masterKey)
	if err != nil {
		return err
	}

	serverMap := make(map[string]dto.KeyDto, len(serverData.Keys))
	for _, key := range serverData.Keys {
		serverMap[key.Filename] = key
	}

	// Apply downloads
	downloadDirPath, err := utils.GetAndCreateSshDirectory(downloadDir)
	if err != nil {
		return err
	}
	if err := applyDownloads(decisions, serverMap, downloadDirPath); err != nil {
		return err
	}

	// Apply uploads
	if len(toUpload) > 0 {
		var multipartBody bytes.Buffer
		multipartWriter := multipart.NewWriter(&multipartBody)
		for _, f := range toUpload {
			w, err := multipartWriter.CreateFormFile("keys[]", f.name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, bytes.NewReader(f.encData)); err != nil {
				return err
			}
		}
		// ssh_config is required by the server. Echo back the server's current
		// config so the field is always present without overwriting remote changes.
		sshConfigJSON, err := json.Marshal(serverData.SshConfig)
		if err != nil {
			return err
		}
		cfgField, err := multipartWriter.CreateFormField("ssh_config")
		if err != nil {
			return err
		}
		if _, err := cfgField.Write(sshConfigJSON); err != nil {
			return err
		}
		multipartWriter.Close()

		client := retrieval.NewRetrievalClient()
		if err = client.UploadData(profile, p, multipartWriter, multipartBody); err != nil {
			return err
		}
	}

	// Write config and known_hosts from server
	if err := utils.WriteConfig(lo.Map(serverData.SshConfig, func(cfg dto.SshConfigDto, _ int) models.Host {
		return models.Host{
			Host:          cfg.Host,
			Values:        cfg.Values,
			IdentityFiles: cfg.IdentityFiles,
		}
	}), downloadDir); err != nil {
		return err
	}

	if len(serverData.KnownHosts) > 0 {
		entries := lo.Map(serverData.KnownHosts, func(kh dto.KnownHostDto, _ int) models.KnownHostEntry {
			return models.KnownHostEntry{
				HostPattern: kh.HostPattern,
				KeyType:     kh.KeyType,
				KeyData:     kh.KeyData,
				Marker:      kh.Marker,
			}
		})
		if err := utils.WriteKnownHosts(entries, downloadDir); err != nil {
			return err
		}
	}

	// Summary
	var nUploaded, nDownloaded, nSkipped int
	for _, d := range decisions {
		switch d.action {
		case "upload":
			nUploaded++
		case "download":
			nDownloaded++
		case "skip":
			nSkipped++
		}
	}
	fmt.Printf("Sync complete: %d uploaded, %d downloaded, %d skipped.\n",
		nUploaded, nDownloaded, nSkipped)
	return nil
}

// buildSyncDecisions compares the local SSH directory against the server's key list and
// returns a decision for each file plus the list of files that need to be uploaded.
// Timestamps are truncated to second precision before comparison to handle filesystems
// (e.g. HFS+) that store mtime at coarser granularity than the server's microseconds.
// When timestamps are equal but content differs, the server version wins.
func buildSyncDecisions(
	localDir string,
	serverKeys []dto.KeyDto,
	masterKey []byte,
) ([]syncDecision, []fileToUpload, error) {
	serverMap := make(map[string]dto.KeyDto, len(serverKeys))
	for _, key := range serverKeys {
		serverMap[key.Filename] = key
	}

	localFiles, err := os.ReadDir(localDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}

	var decisions []syncDecision
	var toUpload []fileToUpload
	localNames := make(map[string]struct{})

	for _, file := range localFiles {
		if file.IsDir() || isSkippedBinaryUpload(file.Name()) || file.Name() == "config" {
			continue
		}

		name := file.Name()
		localNames[name] = struct{}{}

		fileInfo, err := file.Info()
		if err != nil {
			return nil, nil, err
		}
		localMod := fileInfo.ModTime().Truncate(time.Second)

		localContent, err := os.ReadFile(filepath.Join(localDir, name))
		if err != nil {
			return nil, nil, err
		}

		serverKey, onServer := serverMap[name]

		if !onServer {
			encData, err := utils.EncryptWithMasterKey(localContent, masterKey)
			if err != nil {
				return nil, nil, err
			}
			toUpload = append(toUpload, fileToUpload{name: name, encData: encData})
			decisions = append(decisions, syncDecision{filename: name, action: "upload"})
			continue
		}

		// No server timestamp: take server version if content differs, skip if same.
		if serverKey.UpdatedAt == nil {
			if bytes.Equal(localContent, serverKey.Data) {
				decisions = append(decisions, syncDecision{filename: name, action: "skip"})
			} else {
				decisions = append(decisions, syncDecision{filename: name, action: "download"})
			}
			continue
		}

		serverMod := serverKey.UpdatedAt.Truncate(time.Second)

		if serverMod.After(localMod) {
			decisions = append(decisions, syncDecision{filename: name, action: "download"})
		} else if localMod.After(serverMod) {
			encData, err := utils.EncryptWithMasterKey(localContent, masterKey)
			if err != nil {
				return nil, nil, err
			}
			toUpload = append(toUpload, fileToUpload{name: name, encData: encData})
			decisions = append(decisions, syncDecision{filename: name, action: "upload"})
		} else {
			// Timestamps equal to the second. Server wins on differing content so
			// that deliberate mtime manipulation doesn't silently clobber the server.
			if bytes.Equal(localContent, serverKey.Data) {
				decisions = append(decisions, syncDecision{filename: name, action: "skip"})
			} else {
				decisions = append(decisions, syncDecision{filename: name, action: "download"})
			}
		}
	}

	// Keys on server not present locally → download
	for _, serverKey := range serverKeys {
		if isReservedFilename(serverKey.Filename) {
			continue
		}
		if _, exists := localNames[serverKey.Filename]; !exists {
			decisions = append(decisions, syncDecision{filename: serverKey.Filename, action: "download"})
		}
	}

	return decisions, toUpload, nil
}

// applyDownloads writes each server key that was decided for download into downloadDirPath.
func applyDownloads(decisions []syncDecision, serverMap map[string]dto.KeyDto, downloadDirPath string) error {
	for _, d := range decisions {
		if d.action != "download" {
			continue
		}
		serverKey := serverMap[d.filename]
		localPath := filepath.Join(downloadDirPath, d.filename)
		if err := os.WriteFile(localPath, serverKey.Data, 0600); err != nil {
			return err
		}
		if serverKey.UpdatedAt != nil {
			_ = os.Chtimes(localPath, *serverKey.UpdatedAt, *serverKey.UpdatedAt)
		}
	}
	return nil
}
