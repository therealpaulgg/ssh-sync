package models

type Host struct {
	Host         string            `json:"host"`
	IdentityFile string            `json:"identity_file"`
	Values       map[string]string `json:"values"`
}
