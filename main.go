package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/urfave/cli/v2"
)

type Profile struct {
	Username    string `json:"username"`
	MachineName string `json:"machine_name"`
}

func RetrievePrivateKey() (jwk.Key, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := path.Join(user.HomeDir, ".ssh-sync", "keypair")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	key, err := jwk.ParseKey(file, jwk.WithPEM(true))
	return key, err
}

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

type Host struct {
	Host   string            `json:"host"`
	Values map[string]string `json:"values"`
}

func parseSshConfig() ([]Host, error) {
	// parse the ssh config file and return a list of hosts
	// the ssh config file is located at ~/.ssh/config
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := path.Join(user.HomeDir, ".ssh", "config")
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	var hosts []Host
	scanner := bufio.NewScanner(file)
	var currentHost *Host
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Host ") {
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &Host{
				Host:   strings.TrimPrefix(line, "Host "),
				Values: make(map[string]string),
			}
		} else if re := regexp.MustCompile(`^\s+(\w+)[ =](.+)$`); re.Match([]byte(line)) {
			currentHost.Values[re.FindStringSubmatch(line)[1]] = re.FindStringSubmatch(line)[2]
		}
	}
	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}
	return hosts, nil
}

func main() {
	app := &cli.App{
		Name:        "ssh-sync",
		Description: "Syncs your ssh keys to a remote server",
		Commands: []*cli.Command{
			{
				Name:        "setup",
				Description: "Set up your system to use ssh-sync.",
				Action: func(c *cli.Context) error {
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
					// if not:
					// ask user to pick a username.
					fmt.Print("Please enter a username. This will be used to identify your account on the server: ")
					var username string
					_, err = fmt.Scanln(&username)
					if err != nil {
						return err
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
					return nil
				},
			},
			{
				Name: "token",
				Action: func(c *cli.Context) error {
					setup, err := checkIfSetup()
					if err != nil {
						return err
					}
					if !setup {
						fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
						return nil
					}
					key, err := RetrievePrivateKey()
					if err != nil {
						return err
					}
					tok, err := jwt.NewBuilder().Issuer("github.com/therealpaulgg/ssh-sync").IssuedAt(time.Now()).Expiration(time.Now().Add(time.Minute)).Build()
					if err != nil {
						return err
					}
					signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES512, key))
					if err != nil {
						return err
					}
					fmt.Println(string(signed))
					return nil
				},
			},
			{
				Name: "upload",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"p"},
						Usage:   "Path to the ssh keys",
					},
				},
				Action: func(c *cli.Context) error {
					p := c.String("path")
					if p == "" {
						user, err := user.Current()
						if err != nil {
							return err
						}
						p = path.Join(user.HomeDir, ".ssh")
					}
					data, err := os.ReadDir(p)
					if err != nil {
						return err
					}
					for _, file := range data {
						if file.IsDir() {
							continue
						}
						println(file.Name())
					}
					return nil
				},
			},
			{
				Name: "parse-config",
				Action: func(c *cli.Context) error {
					hosts, err := parseSshConfig()
					if err != nil {
						return err
					}
					for _, host := range hosts {
						fmt.Println(host)
					}
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
