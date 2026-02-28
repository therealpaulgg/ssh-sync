package actions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func RotateMasterKey(c *cli.Context) error {
	setup, err := utils.CheckIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}

	fmt.Println("This will generate a new master key and re-encrypt all SSH keys stored on the server.")
	fmt.Println("Other machines will automatically pick up the new master key on their next 'download'.")
	fmt.Println()
	fmt.Print("Continue? (y/n): ")

	scanner := bufio.NewScanner(os.Stdin)
	var answer string
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
		return err
	}
	if answer != "y" {
		fmt.Println("Rotation cancelled.")
		return nil
	}

	fmt.Println("Retrieving current master key...")
	oldMasterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return fmt.Errorf("retrieving master key: %w", err)
	}

	fmt.Println("Generating new master key...")
	newMasterKey, err := createMasterKey()
	if err != nil {
		return fmt.Errorf("generating new master key: %w", err)
	}

	token, err := utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}

	profile, err := utils.GetProfile()
	if err != nil {
		return fmt.Errorf("getting profile: %w", err)
	}

	// Download all existing encrypted SSH keys from the server
	fmt.Println("Downloading existing keys from server...")
	dataUrl := profile.ServerUrl
	dataUrl.Path = "/api/v1/data"
	req, err := http.NewRequest("GET", dataUrl.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}

	var serverData dto.DataDto
	if err := json.NewDecoder(res.Body).Decode(&serverData); err != nil {
		return fmt.Errorf("decoding server data: %w", err)
	}

	// Decrypt each key with old master key, re-encrypt with new master key
	fmt.Println("Re-encrypting SSH keys with new master key...")
	reencryptedKeys := make([]dto.KeyDto, len(serverData.Keys))
	for i, key := range serverData.Keys {
		plaintext, err := utils.DecryptWithMasterKey(key.Data, oldMasterKey)
		if err != nil {
			return fmt.Errorf("decrypting key %q: %w", key.Filename, err)
		}
		ciphertext, err := utils.EncryptWithMasterKey(plaintext, newMasterKey)
		if err != nil {
			return fmt.Errorf("re-encrypting key %q: %w", key.Filename, err)
		}
		reencryptedKeys[i] = dto.KeyDto{
			Filename: key.Filename,
			Data:     ciphertext,
		}
	}

	// Upload re-encrypted keys back to the server
	fmt.Println("Uploading re-encrypted keys to server...")
	token, err = utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}

	var uploadBody bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBody)

	// Re-send SSH config (even if empty, send "[]" so server accepts it)
	sshConfigJSON, err := json.Marshal(serverData.SshConfig)
	if err != nil {
		return fmt.Errorf("marshaling ssh config: %w", err)
	}
	configField, err := uploadWriter.CreateFormField("ssh_config")
	if err != nil {
		return err
	}
	if _, err := configField.Write(sshConfigJSON); err != nil {
		return err
	}

	// Re-send known hosts if any
	if len(serverData.KnownHosts) > 0 {
		khJSON, err := json.Marshal(serverData.KnownHosts)
		if err != nil {
			return fmt.Errorf("marshaling known hosts: %w", err)
		}
		khField, err := uploadWriter.CreateFormField("known_hosts")
		if err != nil {
			return err
		}
		if _, err := khField.Write(khJSON); err != nil {
			return err
		}
	}

	// Write re-encrypted key files
	for _, key := range reencryptedKeys {
		fw, err := uploadWriter.CreateFormFile("keys[]", key.Filename)
		if err != nil {
			return err
		}
		if _, err := fw.Write(key.Data); err != nil {
			return err
		}
	}
	uploadWriter.Close()

	uploadUrl := profile.ServerUrl
	uploadUrl.Path = "/api/v1/data"
	uploadReq, err := http.NewRequest("POST", uploadUrl.String(), &uploadBody)
	if err != nil {
		return err
	}
	uploadReq.Header.Add("Authorization", "Bearer "+token)
	uploadReq.Header.Add("Content-Type", uploadWriter.FormDataContentType())
	uploadRes, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return err
	}
	defer uploadRes.Body.Close()
	if uploadRes.StatusCode != http.StatusOK {
		return errors.New("failed to upload re-encrypted keys. status code: " + strconv.Itoa(uploadRes.StatusCode))
	}

	// Get all machine public keys
	fmt.Println("Fetching machine public keys...")
	token, err = utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}
	pubKeysUrl := profile.ServerUrl
	pubKeysUrl.Path = "/api/v1/machines/public-keys"
	pkReq, err := http.NewRequest("GET", pubKeysUrl.String(), nil)
	if err != nil {
		return err
	}
	pkReq.Header.Add("Authorization", "Bearer "+token)
	pkRes, err := http.DefaultClient.Do(pkReq)
	if err != nil {
		return err
	}
	defer pkRes.Body.Close()
	if pkRes.StatusCode != http.StatusOK {
		return errors.New("failed to get machine public keys. status code: " + strconv.Itoa(pkRes.StatusCode))
	}

	var machinesDto dto.MachinesPublicKeysDto
	if err := json.NewDecoder(pkRes.Body).Decode(&machinesDto); err != nil {
		return fmt.Errorf("decoding machine public keys: %w", err)
	}

	// Encrypt new master key for each machine
	fmt.Printf("Distributing new master key to %d machine(s)...\n", len(machinesDto.Machines))
	perMachineKeys := make([]dto.PerMachineMasterKeyDto, 0, len(machinesDto.Machines))
	for _, m := range machinesDto.Machines {
		var encKey []byte
		if len(m.EncapsulationKey) > 0 {
			encKey, err = utils.EncryptWithPQPublicKey(newMasterKey, m.EncapsulationKey)
			if err != nil {
				return fmt.Errorf("encrypting master key for machine %q (PQ): %w", m.Name, err)
			}
		} else {
			encKey, err = utils.EncryptWithECPublicKey(newMasterKey, m.PublicKey)
			if err != nil {
				return fmt.Errorf("encrypting master key for machine %q (EC): %w", m.Name, err)
			}
		}
		perMachineKeys = append(perMachineKeys, dto.PerMachineMasterKeyDto{
			MachineID:          m.MachineID,
			EncryptedMasterKey: encKey,
		})
	}

	// Post rotation entries to server
	rotationReq := dto.MasterKeyRotationRequestDto{Keys: perMachineKeys}
	rotationBody, err := json.Marshal(rotationReq)
	if err != nil {
		return fmt.Errorf("marshaling rotation request: %w", err)
	}

	token, err = utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}
	rotUrl := profile.ServerUrl
	rotUrl.Path = "/api/v1/key-rotation"
	rotHttpReq, err := http.NewRequest("POST", rotUrl.String(), bytes.NewReader(rotationBody))
	if err != nil {
		return err
	}
	rotHttpReq.Header.Add("Authorization", "Bearer "+token)
	rotHttpReq.Header.Add("Content-Type", "application/json")
	rotHttpRes, err := http.DefaultClient.Do(rotHttpReq)
	if err != nil {
		return err
	}
	defer rotHttpRes.Body.Close()
	if rotHttpRes.StatusCode != http.StatusOK {
		return errors.New("failed to post key rotation. status code: " + strconv.Itoa(rotHttpRes.StatusCode))
	}

	// Save new master key locally (encrypted with this machine's key)
	fmt.Println("Saving new master key locally...")
	encryptedNew, err := utils.Encrypt(newMasterKey)
	if err != nil {
		return fmt.Errorf("encrypting new master key for local storage: %w", err)
	}
	if err := saveMasterKey(encryptedNew); err != nil {
		return fmt.Errorf("saving new master key: %w", err)
	}

	fmt.Println()
	fmt.Println("Master key rotation complete!")
	fmt.Println("Other machines will automatically apply the new key on their next 'download'.")
	return nil
}
