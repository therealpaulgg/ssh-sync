package models

import "net/url"

type Profile struct {
	Username    string  `json:"username"`
	MachineName string  `json:"machine_name"`
	ServerUrl   url.URL `json:"server_url"`
}
