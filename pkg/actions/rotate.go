package actions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	fmt.Println("Your local SSH files must be up to date before rotating.")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Run 'download' now to sync local files before rotating? (y/n): ")
	var downloadFirst string
	if err := utils.ReadLineFromStdin(scanner, &downloadFirst); err != nil {
		return err
	}
	if downloadFirst == "y" {
		if err := Download(c); err != nil {
			return fmt.Errorf("downloading before rotation: %w", err)
		}
	}

	fmt.Print("Continue with master key rotation? (y/n): ")
	var answer string
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
		return err
	}
	if answer != "y" {
		fmt.Println("Rotation cancelled.")
		return nil
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

	fmt.Println("Fetching machine public keys...")
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

	// Encrypt the new master key for every machine, including this one.
	// This machine will consume its own entry via applyPendingRotation inside Upload.
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

	// Upload picks up this machine's own rotation entry via applyPendingRotation,
	// saves the new master key locally, then re-encrypts all local SSH files and
	// pushes them to the server.
	fmt.Println("Re-encrypting and uploading local keys with new master key...")
	if err := Upload(c); err != nil {
		return fmt.Errorf("uploading re-encrypted keys: %w", err)
	}

	fmt.Println()
	fmt.Println("Master key rotation complete!")
	fmt.Println("Other machines will automatically apply the new key on their next 'download'.")
	return nil
}
