package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"

	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", "http://localhost:3000/api/v1/data", nil)
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
	var dataDto DataDto
	err = json.NewDecoder(res.Body).Decode(&dataDto)
	if err != nil {
		return err
	}
	masterKey, err := utils.Decrypt(dataDto.MasterKey)
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
		p = path.Join(user.HomeDir, ".ssh")
	}
	data, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	for _, file := range data {
		if file.IsDir() || file.Name() == "config" || file.Name() == "authorized_keys" {
			continue
		}
		f, err := os.OpenFile(path.Join(p, file.Name()), os.O_RDONLY, 0600)
		if err != nil {
			return err
		}
		// read file into buffer
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		encBytes, err := utils.EncryptWithMasterKey(data, masterKey)
		if err != nil {
			return err
		}
		w, _ := multipartWriter.CreateFormFile("keys[]", file.Name())
		_, err = io.Copy(w, bytes.NewReader(encBytes))
		if err != nil {
			return err
		}
	}
	multipartWriter.Close()
	req2, err := http.NewRequest("POST", "http://localhost:3000/api/v1/data", &multipartBody)
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
