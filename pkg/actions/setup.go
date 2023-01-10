package actions

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
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
	p := filepath.Join(user.HomeDir, ".ssh-sync", "profile.json")
	_, err = os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func generateKey() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pub := &priv.PublicKey
	if err != nil {
		return nil, nil, err
	}
	// then the program will save the keypair to ~/.ssh-sync/keypair.pub and ~/.ssh-sync/keypair
	user, err := user.Current()
	if err != nil {
		return nil, nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync")
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, err
	}
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	pubOut, err := os.OpenFile(filepath.Join(p, "keypair.pub"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, nil, err
	}
	defer pubOut.Close()
	if err := pem.Encode(pubOut, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}); err != nil {
		return nil, nil, err
	}
	privOut, err := os.OpenFile(filepath.Join(p, "keypair"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, nil, err
	}
	defer privOut.Close()
	if err := pem.Encode(privOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

func getProfile() (*models.Profile, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "profile.json")
	dat, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var profile models.Profile
	if err = json.Unmarshal(dat, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func saveProfile(username string, machineName string, serverUrl url.URL) error {
	// then the program will save the profile to ~/.ssh-sync/profile.json
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "profile.json")
	profile := models.Profile{
		Username:    username,
		MachineName: machineName,
		ServerUrl:   serverUrl,
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

func checkIfAccountExists(username string, serverUrl *url.URL) (bool, error) {
	url := *serverUrl
	url.Path = "/api/v1/users/" + username
	res, err := http.Get(url.String())
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
	pubkeyFile, err := os.Open(filepath.Join(user.HomeDir, ".ssh-sync", "keypair.pub"))
	if err != nil {
		return nil, err
	}
	return pubkeyFile, nil
}

func createMasterKey() ([]byte, error) {
	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	if err != nil {
		return nil, err
	}
	return masterKey, nil
}

func newAccountSetup(serverUrl *url.URL) error {
	// ask user to pick a username.
	fmt.Print("Please enter a username. This will be used to identify your account on the server: ")
	var username string
	_, err := fmt.Scanln(&username)
	if err != nil {
		return err
	}
	exists, err := checkIfAccountExists(username, serverUrl)
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
	_, _, err = generateKey()
	if err != nil {
		return err
	}
	// then the program will save the profile to ~/.ssh-sync/profile.json
	err = saveProfile(username, machineName, *serverUrl)
	if err != nil {
		return err
	}
	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)
	pubkeyFile, err := getPubkeyFile()
	if err != nil {
		return err
	}
	masterKey, err := createMasterKey()
	if err != nil {
		return err
	}
	encryptedMasterKey, err := utils.Encrypt(masterKey)
	if err != nil {
		return err
	}
	fileWriter, _ := multipartWriter.CreateFormFile("key", pubkeyFile.Name())
	io.Copy(fileWriter, pubkeyFile)
	multipartWriter.WriteField("username", username)
	multipartWriter.WriteField("machine_name", machineName)
	multipartWriter.WriteField("master_key", string(encryptedMasterKey))
	multipartWriter.Close()
	setupUrl := *serverUrl
	setupUrl.Path = "/api/v1/setup"
	req, err := http.NewRequest("POST", setupUrl.String(), &multipartBody)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("failed to create user. status code: " + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func existingAccountSetup(serverUrl *url.URL) error {
	fmt.Print("Please enter a username. This will be used to identify your account on the server: ")
	var username string
	_, err := fmt.Scanln(&username)
	if err != nil {
		return err
	}
	exists, err := checkIfAccountExists(username, serverUrl)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user doesn't exist. try creating a new account")
	}
	fmt.Print("Please enter a name for this machine: ")
	var machineName string
	_, err = fmt.Scanln(&machineName)
	if err != nil {
		return err
	}
	wsUrl := *serverUrl
	if wsUrl.Scheme == "http" {
		wsUrl.Scheme = "ws"
	} else {
		wsUrl.Scheme = "wss"
	}
	wsUrl.Path = "/api/v1/setup/existing"
	dialer := ws.Dialer{}
	conn, _, _, err := dialer.Dial(context.Background(), wsUrl.String())
	if err != nil {
		return err
	}
	defer conn.Close()
	dto := dto.UserMachineDto{}
	dto.Username = username
	dto.MachineName = machineName
	b, err := json.Marshal(dto)
	if err != nil {
		return err
	}
	err = wsutil.WriteClientBinary(conn, b)
	if err != nil {
		return err
	}
	challengePhrase, err := wsutil.ReadServerBinary(conn)
	if err != nil {
		return err
	}
	fmt.Println(string(challengePhrase))
	waiting, err := wsutil.ReadServerBinary(conn)
	if err != nil {
		return err
	}
	fmt.Println(string(waiting))
	fmt.Println("here 1")
	fmt.Println("Generating keypair...")
	_, _, err = generateKey()
	if err != nil {
		return err
	}
	saveProfile(username, machineName, *serverUrl)
	f, err := getPubkeyFile()
	if err != nil {
		return err
	}
	defer f.Close()
	pubkey, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = wsutil.WriteClientBinary(conn, pubkey)
	if err != nil {
		return err
	}
	waiting, err = wsutil.ReadServerBinary(conn)
	if err != nil {
		return err
	}
	fmt.Println(string(waiting))
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
	fmt.Print("Please enter your server address (http/https): ")
	var serverAddress string
	_, err = fmt.Scanln(&serverAddress)
	if err != nil {
		return err
	}
	serverUrl, err := url.Parse(serverAddress)
	if err != nil {
		return err
	} else if serverUrl.Scheme == "" || serverUrl.Host == "" {
		return errors.New("invalid server address")
	} else if serverUrl.Scheme != "http" && serverUrl.Scheme != "https" {
		return errors.New("server must use http or https")
	}
	if serverUrl.Scheme == "http" {
		fmt.Println("WARNING: Your server is using HTTP. This is not secure. You should use HTTPS.")
	}
	// test connection to server
	_, err = http.Get(serverUrl.String())
	if err != nil {
		return err
	}
	fmt.Print("Do you already have an account on the ssh-sync server? (y/n): ")
	var answer string
	_, err = fmt.Scanln(&answer)
	if err != nil {
		return err
	}
	if answer == "y" {
		return existingAccountSetup(serverUrl)
	}
	return newAccountSetup(serverUrl)
}
