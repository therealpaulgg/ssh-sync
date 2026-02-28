package actions

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func applyPendingRotation(profile *models.Profile) error {
	token, err := utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}

	rotUrl := profile.ServerUrl
	rotUrl.Path = "/api/v1/key-rotation"
	req, err := http.NewRequest("GET", rotUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		// No pending rotation
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected status checking key rotation: " + res.Status)
	}

	var rotDto dto.EncryptedMasterKeyDto
	if err := json.NewDecoder(res.Body).Decode(&rotDto); err != nil {
		return fmt.Errorf("decoding key rotation response: %w", err)
	}

	fmt.Println("Applying pending master key rotation...")
	newMasterKey, err := utils.Decrypt(rotDto.EncryptedMasterKey)
	if err != nil {
		return fmt.Errorf("decrypting rotated master key: %w", err)
	}

	encryptedNew, err := utils.Encrypt(newMasterKey)
	if err != nil {
		return fmt.Errorf("re-encrypting new master key for local storage: %w", err)
	}
	if err := saveMasterKey(encryptedNew); err != nil {
		return fmt.Errorf("saving rotated master key: %w", err)
	}

	// Acknowledge the rotation so the server can clean it up
	token, err = utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token for delete: %w", err)
	}
	delUrl := profile.ServerUrl
	delUrl.Path = "/api/v1/key-rotation"
	delReq, err := http.NewRequest("DELETE", delUrl.String(), nil)
	if err != nil {
		return err
	}
	delReq.Header.Add("Authorization", "Bearer "+token)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		return err
	}
	defer delRes.Body.Close()
	if delRes.StatusCode != http.StatusOK {
		return errors.New("failed to acknowledge key rotation: " + delRes.Status)
	}

	fmt.Println("Master key rotation applied successfully.")
	return nil
}

func Download(c *cli.Context) error {
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

	client := retrieval.NewRetrievalClient()
	data, err := client.GetUserData(profile)
	if err != nil {
		return err
	}
	var directory string
	if p := c.String("path"); p != "" {
		directory = p
	} else if c.Bool("safe-mode") {
		fmt.Println("Executing in safe mode (keys writing to .ssh-sync-data)")
		directory = ".ssh-sync-data"
	} else {
		directory = ".ssh"
	}
	if err := utils.WriteConfig(lo.Map(data.SshConfig, func(config dto.SshConfigDto, i int) models.Host {
		return models.Host{
			Host:          config.Host,
			Values:        config.Values,
			IdentityFiles: config.IdentityFiles,
		}
	}), directory); err != nil {
		return err
	}
	for _, key := range data.Keys {
		if isReservedFilename(key.Filename) {
			continue
		}
		if err := utils.WriteKey(key.Data, key.Filename, directory); err != nil {
			return err
		}
	}
	if len(data.KnownHosts) > 0 {
		entries := lo.Map(data.KnownHosts, func(kh dto.KnownHostDto, _ int) models.KnownHostEntry {
			return models.KnownHostEntry{
				HostPattern: kh.HostPattern,
				KeyType:     kh.KeyType,
				KeyData:     kh.KeyData,
				Marker:      kh.Marker,
			}
		})
		if err := utils.WriteKnownHosts(entries, directory); err != nil {
			return err
		}
	}

	err = checkForDeletedKeys(data.Keys, directory)

	if err != nil {
		return err
	}
	fmt.Println("Successfully downloaded keys.")
	return nil
}

func checkForDeletedKeys(keys []dto.KeyDto, directory string) error {
	sshDir, err := utils.GetAndCreateSshDirectory(directory)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(sshDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isReservedFilename(d.Name()) {
			return nil
		}
		_, exists := lo.Find(keys, func(key dto.KeyDto) bool {
			return key.Filename == d.Name()
		})
		if exists {
			return nil
		}
		fmt.Printf("Key %s detected on your filesystem that is not in the database. Delete? (y/n): ", d.Name())
		var answer string
		scanner := bufio.NewScanner(os.Stdin)
		if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
			return err
		}
		if answer == "y" {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// isReservedFilename reports whether a filename should never be treated as a
// synced key — either because it is managed separately (config, known_hosts)
// or because it must never leave the local machine (authorized_keys).
func isReservedFilename(name string) bool {
	switch name {
	case "known_hosts", "authorized_keys", "config":
		return true
	}
	return false
}
