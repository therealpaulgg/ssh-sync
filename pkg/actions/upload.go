package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/therealpaulgg/ssh-sync/pkg/models"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
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
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	url := profile.ServerUrl
	url.Path = "/api/v1/data"
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New("failed to get data. status code: " + strconv.Itoa(res.StatusCode))
	}
	masterKey, err := utils.RetrieveMasterKey()
	if err != nil {
		return err
	}
	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)
	p := c.String("path")
	if p == "" {
		user, err := user.Current()
		if err != nil {
			return err
		}
		p = filepath.Join(user.HomeDir, ".ssh")
	}
	data, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	hosts := []models.Host{}
	for _, file := range data {
		if file.IsDir() || file.Name() == "authorized_keys" {
			continue
		} else if file.Name() == "config" {
			hosts, err = utils.ParseConfig()
			if err != nil {
				return err
			}
			if len(hosts) == 0 {
				return errors.New("your ssh config is empty. Please add some hosts to your ssh config so data can be uploaded.")
			}
			continue
		} else if file.Name() == "known_hosts" {
			f, err := os.OpenFile(filepath.Join(p, file.Name()), os.O_RDONLY, 0600)
			if err != nil {
				return err
			}
			// read file into buffer
			knownHostsData, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			encBytes, err := utils.EncryptWithMasterKey(knownHostsData, masterKey)
			if err != nil {
				return err
			}
			w, err := multipartWriter.CreateFormField("known_hosts")
			if err != nil {
				return err
			}
			if _, err := w.Write(encBytes); err != nil {
				return err
			}
			f.Close()
			continue
		}
		f, err := os.OpenFile(filepath.Join(p, file.Name()), os.O_RDONLY, 0600)
		if err != nil {
			return err
		}
		// read file into buffer
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		encBytes, err := utils.EncryptWithMasterKey(data, masterKey)
		if err != nil {
			return err
		}
		w, _ := multipartWriter.CreateFormFile("keys[]", file.Name())
		if _, err := io.Copy(w, bytes.NewReader(encBytes)); err != nil {
			return err
		}
	}
	if hosts != nil {
		jsonBytes, err := json.Marshal(hosts)
		if err != nil {
			return err
		}
		w, err := multipartWriter.CreateFormField("ssh_config")
		if err != nil {
			return err
		}
		if _, err := w.Write(jsonBytes); err != nil {
			return err
		}
	}
	multipartWriter.Close()
	url2 := profile.ServerUrl
	url2.Path = "/api/v1/data"
	req2, err := http.NewRequest("POST", url2.String(), &multipartBody)
	if err != nil {
		return err
	}
	req2.Header.Add("Authorization", "Bearer "+token)
	req2.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return err
	}
	if res2.StatusCode != 200 {
		return errors.New("failed to upload data. status code: " + strconv.Itoa(res2.StatusCode))
	}
	fmt.Println("Successfully uploaded keys.")
	return nil
}
