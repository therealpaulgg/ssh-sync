package models

type Host struct {
	Host          string              `json:"host"`
	IdentityFiles []string            `json:"identity_files"`
	Values        map[string][]string `json:"values"`
}
