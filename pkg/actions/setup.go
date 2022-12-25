package actions

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"

	"github.com/urfave/cli/v2"
)

func checkIfSetup() (bool, error) {
	// check if ~/.ssh-sync/profile.json exists
	// if it does, return true
	// if it doesn't, return false
	user, err := user.Current()
	if err != nil {
		return false, err
	}
	p := path.Join(user.HomeDir, ".ssh-sync", "profile.json")
	_, err = os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type Profile struct {
	Username    string `json:"username"`
	MachineName string `json:"machine_name"`
}

func generateKey() error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pub := &priv.PublicKey
	if err != nil {
		return err
	}
	// then the program will save the keypair to ~/.ssh-sync/keypair.pub and ~/.ssh-sync/keypair
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := path.Join(user.HomeDir, ".ssh-sync")
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	pubOut, err := os.OpenFile(path.Join(p, "keypair.pub"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer pubOut.Close()
	if err := pem.Encode(pubOut, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}); err != nil {
		return err
	}
	privOut, err := os.OpenFile(path.Join(p, "keypair"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer privOut.Close()
	if err := pem.Encode(privOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}
	return nil
}

func saveProfile(username string, machineName string) error {
	// then the program will save the profile to ~/.ssh-sync/profile.json
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := path.Join(user.HomeDir, ".ssh-sync", "profile.json")
	profile := Profile{
		Username:    username,
		MachineName: machineName,
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	err = os.WriteFile(p, profileBytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

func checkIfAccountExists(username string) (bool, error) {
	res, err := http.Get("http://localhost:3000/api/v1/users/" + username)
	if err != nil {
		return false, err
	}
	if res.StatusCode == 404 {
		return false, nil
	}
	return true, nil
}

func getPubkeyFile() (*os.File, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	pubkeyFile, err := os.Open(path.Join(user.HomeDir, ".ssh-sync", "keypair.pub"))
	if err != nil {
		return nil, err
	}
	return pubkeyFile, nil
}

func newAccountSetup() error {
	// ask user to pick a username.
	fmt.Print("Please enter a username. This will be used to identify your account on the server: ")
	var username string
	_, err := fmt.Scanln(&username)
	if err != nil {
		return err
	}
	exists, err := checkIfAccountExists(username)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user already exists on the server")
	}
	// ask user to pick a name for this machine (default to current system name)
	fmt.Print("Please enter a name for this machine: ")
	var machineName string
	_, err = fmt.Scanln(&machineName)
	if err != nil {
		return err
	}
	// then the program will generate a keypair, and upload the public key to the server
	fmt.Println("Generating keypair...")
	err = generateKey()
	if err != nil {
		return err
	}
	// then the program will save the profile to ~/.ssh-sync/profile.json
	saveProfile(username, machineName)
	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)
	pubkeyFile, err := getPubkeyFile()
	if err != nil {
		return err
	}
	multipartWriter.CreateFormFile("key", pubkeyFile.Name())
	multipartWriter.WriteField("username", username)
	multipartWriter.WriteField("machine_name", machineName)
	multipartWriter.Close()
	req, err := http.Post("http://localhost:3000/api/v1/user/", "multipart/form-data", &multipartBody)
	if err != nil {
		return err
	}
	if req.StatusCode != 200 {
		return errors.New("failed to create user. status code: " + strconv.Itoa(req.StatusCode))
	}
	return nil
}

func existingAccountSetup() error {
	// ask user to enter their username
	fmt.Print("Please enter your username: ")
	var username string
	_, err := fmt.Scanln(&username)
	if err != nil {
		return err
	}
	exists, err := checkIfAccountExists(username)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user does not exist on the server")
	}
	// ask user to enter their machine name
	fmt.Print("Please enter the name of this machine: ")
	var machineName string
	_, err = fmt.Scanln(&machineName)
	if err != nil {
		return err
	}
	// then the program will generate a keypair, and upload the public key to the server
	fmt.Println("Generating keypair...")
	err = generateKey()
	if err != nil {
		return err
	}
	// then the program will save the profile to ~/.ssh-sync/profile.json
	saveProfile(username, machineName)
	// TODO challenge
	return nil
}

func Setup(c *cli.Context) error {
	// all files will be stored in ~/.ssh-sync
	// there will be a profile.json file containing the machine name and the username
	// there will also be a keypair.
	// check if setup has been completed before
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if setup {
		// if it has been completed, the user may want to restart.
		// if so this is a destructive operation and will result in the deletion of all saved data relating to ssh-sync.
		fmt.Println("ssh-sync has already been set up on this system.")
		return nil
	}
	// ask user if they already have an account on the ssh-sync server.
	fmt.Print("Do you already have an account on the ssh-sync server? (y/n): ")
	var answer string
	_, err = fmt.Scanln(&answer)
	if err != nil {
		return err
	}
	if answer == "y" {
		return existingAccountSetup()
	}
	return newAccountSetup()
}
