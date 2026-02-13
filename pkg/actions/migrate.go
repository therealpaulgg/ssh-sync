package actions

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Migrate(c *cli.Context) error {
	setup, err := utils.CheckIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}

	// Check current key format
	format, err := utils.DetectKeyFormat()
	if err != nil {
		return fmt.Errorf("detecting key format: %w", err)
	}
	if format == utils.FormatPostQuantum {
		fmt.Println("Your keys are already using post-quantum cryptography (ML-DSA-65 + ML-KEM-768). No migration needed.")
		return nil
	}

	fmt.Println("This will migrate your keys from classical ECDSA/ECDH-ES to post-quantum")
	fmt.Println("cryptography (ML-DSA-65 for signatures, ML-KEM-768 for key encapsulation).")
	fmt.Println()
	fmt.Println("What this does:")
	fmt.Println("  1. Decrypt your master key using the current EC keypair")
	fmt.Println("  2. Generate new post-quantum keypairs")
	fmt.Println("  3. Re-encrypt your master key with the new ML-KEM-768 key")
	fmt.Println("  4. Upload the new public key to the server")
	fmt.Println()
	fmt.Println("Your encrypted SSH keys on the server remain unchanged (AES-256-GCM")
	fmt.Println("is already quantum-resistant). Only the key wrapping is upgraded.")
	fmt.Println()
	fmt.Print("Continue? (y/n): ")

	scanner := bufio.NewScanner(os.Stdin)
	var answer string
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
		return err
	}
	if answer != "y" {
		fmt.Println("Migration cancelled.")
		return nil
	}

	// Step 1: Decrypt the master key using the legacy EC key
	fmt.Println("Decrypting master key with current EC keypair...")
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return fmt.Errorf("retrieving master key: %w", err)
	}

	// Step 2: Get a JWT token signed with the OLD key BEFORE we overwrite it.
	// The server still has our EC public key and will verify against it.
	fmt.Println("Authenticating with server using current EC key...")
	token, err := utils.GetToken()
	if err != nil {
		return fmt.Errorf("getting auth token: %w", err)
	}

	// Step 3: Back up old key files
	u, err := user.Current()
	if err != nil {
		return err
	}
	sshSyncDir := filepath.Join(u.HomeDir, ".ssh-sync")

	if err := backupFile(filepath.Join(sshSyncDir, "keypair"), filepath.Join(sshSyncDir, "keypair.bak")); err != nil {
		return fmt.Errorf("backing up keypair: %w", err)
	}
	if err := backupFile(filepath.Join(sshSyncDir, "keypair.pub"), filepath.Join(sshSyncDir, "keypair.pub.bak")); err != nil {
		return fmt.Errorf("backing up keypair.pub: %w", err)
	}
	if err := backupFile(filepath.Join(sshSyncDir, "master_key"), filepath.Join(sshSyncDir, "master_key.bak")); err != nil {
		return fmt.Errorf("backing up master_key: %w", err)
	}

	// Step 4: Generate new post-quantum keypair
	fmt.Println("Generating post-quantum keypair (ML-DSA-65 + ML-KEM-768)...")
	if err := generateKey(); err != nil {
		rollbackMigration(sshSyncDir)
		return fmt.Errorf("generating PQ keys: %w", err)
	}

	// Step 5: Re-encrypt master key with the new ML-KEM-768 key
	fmt.Println("Re-encrypting master key with ML-KEM-768...")
	encryptedMasterKey, err := utils.Encrypt(masterKey)
	if err != nil {
		rollbackMigration(sshSyncDir)
		return fmt.Errorf("encrypting master key with PQ key: %w", err)
	}
	if err := saveMasterKey(encryptedMasterKey); err != nil {
		rollbackMigration(sshSyncDir)
		return fmt.Errorf("saving re-encrypted master key: %w", err)
	}

	// Step 6: Upload the new public key to the server using the pre-obtained token
	fmt.Println("Uploading new public key to server...")
	if err := uploadMigratedKey(token); err != nil {
		rollbackMigration(sshSyncDir)
		return fmt.Errorf("uploading new public key: %w", err)
	}

	// Step 7: Clean up backups
	os.Remove(filepath.Join(sshSyncDir, "keypair.bak"))
	os.Remove(filepath.Join(sshSyncDir, "keypair.pub.bak"))
	os.Remove(filepath.Join(sshSyncDir, "master_key.bak"))

	fmt.Println()
	fmt.Println("Migration complete! Your keys are now using post-quantum cryptography.")
	fmt.Println("  Signing:    ML-DSA-65 (FIPS 204)")
	fmt.Println("  Encryption: ML-KEM-768 (FIPS 203)")
	return nil
}

// uploadMigratedKey sends the new public key to the server via PUT /api/v1/machines/key.
// It uses a pre-obtained token (signed with the old key) since the server still has
// the old public key at this point.
func uploadMigratedKey(token string) error {
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}

	u, err := user.Current()
	if err != nil {
		return err
	}
	pubkeyPath := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
	pubkeyFile, err := os.Open(pubkeyPath)
	if err != nil {
		return err
	}
	defer pubkeyFile.Close()

	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)
	fileWriter, err := multipartWriter.CreateFormFile("key", pubkeyFile.Name())
	if err != nil {
		return err
	}
	if _, err := io.Copy(fileWriter, pubkeyFile); err != nil {
		return err
	}
	multipartWriter.Close()

	url := profile.ServerUrl
	url.Path = "/api/v1/machines/key"
	req, err := http.NewRequest("PUT", url.String(), &multipartBody)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("failed to upload new public key. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func backupFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// rollbackMigration restores backup files if migration fails.
func rollbackMigration(sshSyncDir string) {
	fmt.Fprintln(os.Stderr, "Migration failed, rolling back...")
	restoreBackup(filepath.Join(sshSyncDir, "keypair.bak"), filepath.Join(sshSyncDir, "keypair"))
	restoreBackup(filepath.Join(sshSyncDir, "keypair.pub.bak"), filepath.Join(sshSyncDir, "keypair.pub"))
	restoreBackup(filepath.Join(sshSyncDir, "master_key.bak"), filepath.Join(sshSyncDir, "master_key"))
	os.Remove(filepath.Join(sshSyncDir, "keypair.bak"))
	os.Remove(filepath.Join(sshSyncDir, "keypair.pub.bak"))
	os.Remove(filepath.Join(sshSyncDir, "master_key.bak"))
}

func restoreBackup(backupPath, originalPath string) {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return
	}
	_ = os.WriteFile(originalPath, data, 0600)
}
